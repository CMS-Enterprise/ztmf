package model

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validTiers is the authoritative tier-name enum the API serves. Tests
// assert membership rather than exact values where the underlying data
// is dynamic, and exact values where a transaction-rolled-back fixture
// pins the input.
var validTiers = map[string]bool{
	"Optimal":      true,
	"Advanced":     true,
	"Initial":      true,
	"Traditional":  true,
	"Not Assessed": true,
}

// integrationTestPrefix is the marker every test-synthesized datacall
// carries in its name. Used to identify rows for cleanup, including a
// defensive sweep at test entry that catches anything a previous
// interrupted run left behind. Keep it underscore-separated so a LIKE
// pattern is unambiguous.
const integrationTestPrefix = "integration_test_"

// purgeIntegrationTestRows wipes any datacalls (and their cascaded
// scores via FK ON DELETE CASCADE) whose name carries the integration
// test prefix. Run at the start of each integration test as a defensive
// belt-and-suspenders against cleanup that failed to run on a previous
// invocation. Cheap.
func purgeIntegrationTestRows(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("purge: db.Conn: %v", err)
	}
	defer conn.Release()
	_, err = conn.Exec(ctx,
		`DELETE FROM datacalls WHERE datacall LIKE $1`,
		integrationTestPrefix+"%",
	)
	if err != nil {
		t.Fatalf("purge: delete: %v", err)
	}
}

// TestFindScoresAggregateIntegration runs the real SQL against the
// configured database and verifies the response shape and per-row
// invariants. Catches SQL math regressions (wrong +1 shift, wrong AVG
// window, wrong COALESCE behavior) that unit tests over pure Go cannot
// see.
//
// Requires DB_* env vars (DB_ENDPOINT, DB_USER, etc.) pointing at a
// running Postgres with seeded ZTMF schema and at least one scored
// system. The dev compose stack (port 54321) satisfies this. The
// ephemeral test compose stack on port 8090 also works during the e2e
// step of make test-full. Skipped under `go test -short`.
func TestFindScoresAggregateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	t.Run("ShapeInvariants_AllAggregates", func(t *testing.T) {
		// Pull every aggregate from the DB. Don't filter — we want to
		// validate every row returned.
		aggs, err := FindScoresAggregate(ctx, FindScoresInput{
			IncludePillars: boolPtr(true),
		})
		require.NoError(t, err)
		require.NotEmpty(t, aggs, "expected at least one aggregate from seeded data")

		for _, a := range aggs {
			assert.Greater(t, a.DataCallID, int32(0), "datacallid set")
			assert.Greater(t, a.FismaSystemID, int32(0), "fismasystemid set")
			assert.True(t, a.SystemScore >= 1.0 && a.SystemScore <= 5.0,
				"systemscore in HHS range: %v for system %d / datacall %d",
				a.SystemScore, a.FismaSystemID, a.DataCallID)
			assert.True(t, validTiers[a.SystemTier],
				"systemtier in enum: %q for system %d / datacall %d",
				a.SystemTier, a.FismaSystemID, a.DataCallID)
			// The tier predicate is the single source of truth. Either
			// SQL and Go agree, or this test fails and surfaces the
			// drift immediately.
			assert.Equal(t, Tier(a.SystemScore), a.SystemTier,
				"systemtier matches Tier(systemscore) for system %d / datacall %d",
				a.FismaSystemID, a.DataCallID)

			require.NotEmpty(t, a.PillarScores,
				"include_pillars=true must return pillar entries for system %d / datacall %d",
				a.FismaSystemID, a.DataCallID)

			for _, p := range a.PillarScores {
				assert.Greater(t, p.PillarID, int32(0))
				assert.NotEmpty(t, p.Pillar)
				assert.True(t, p.Score >= 1.0 && p.Score <= 5.0,
					"pillar %s score in HHS range: %v", p.Pillar, p.Score)
				assert.True(t, validTiers[p.Tier],
					"pillar %s tier in enum: %q", p.Pillar, p.Tier)
				assert.Equal(t, Tier(p.Score), p.Tier,
					"pillar %s tier matches Tier(score)", p.Pillar)
			}
		}
	})

	t.Run("IncludePillars_FalseOmitsBreakdown", func(t *testing.T) {
		aggs, err := FindScoresAggregate(ctx, FindScoresInput{})
		require.NoError(t, err)
		require.NotEmpty(t, aggs)
		for _, a := range aggs {
			assert.Empty(t, a.PillarScores,
				"pillarscores must be omitted when IncludePillars is nil/false")
			assert.NotEmpty(t, a.SystemTier, "systemtier still populated")
			assert.True(t, a.SystemScore >= 1.0 && a.SystemScore <= 5.0,
				"systemscore still on HHS scale")
		}
	})

	t.Run("FilterByFismaSystem_ReturnsOnlyThatSystem", func(t *testing.T) {
		// Pick a system that has scores so the filter has something to
		// return. The first aggregate in the unfiltered list is a safe
		// pick.
		all, err := FindScoresAggregate(ctx, FindScoresInput{})
		require.NoError(t, err)
		require.NotEmpty(t, all)
		target := all[0].FismaSystemID

		filtered, err := FindScoresAggregate(ctx, FindScoresInput{
			FismaSystemID: &target,
		})
		require.NoError(t, err)
		require.NotEmpty(t, filtered)
		for _, a := range filtered {
			assert.Equal(t, target, a.FismaSystemID,
				"filter must scope to requested system")
		}
	})

	t.Run("FilterByUnscoredSystem_ReturnsEmpty", func(t *testing.T) {
		// 2_000_000 is well above any seeded ID and will never have
		// scores. Aggregate must be empty (not error, not a placeholder
		// "Not Assessed" row for a system that doesn't appear in the
		// scores table at all).
		unscored := int32(2_000_000)
		aggs, err := FindScoresAggregate(ctx, FindScoresInput{
			FismaSystemID: &unscored,
		})
		require.NoError(t, err)
		assert.Empty(t, aggs,
			"unscored system must not appear in /scores/aggregate (frontend cross-references /fismasystems for those)")
	})
}

// TestScoreSaveValidationIntegration covers the validate() guards on
// the write path: notes length and past-deadline rules. Any synthesized
// datacalls carry the integrationTestPrefix and are removed by both a
// startup sweep and explicit cleanup using a fresh connection so a
// pre-closed test connection cannot silently swallow the delete.
func TestScoreSaveValidationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	// Find a datacall whose deadline is in the past, plus one whose
	// deadline is in the future. The seeded datacalls do not guarantee
	// either, so we use the most-recent and oldest as proxies.
	var pastDataCallID, futureDataCallID int32
	err = conn.QueryRow(ctx, `
		SELECT datacallid FROM datacalls
		WHERE deadline < NOW() ORDER BY deadline DESC LIMIT 1
	`).Scan(&pastDataCallID)
	require.NoError(t, err, "need at least one past-deadline datacall in seeded data")

	err = conn.QueryRow(ctx, `
		SELECT datacallid FROM datacalls
		WHERE deadline > NOW() ORDER BY deadline ASC LIMIT 1
	`).Scan(&futureDataCallID)
	if err != nil {
		// If no future-deadline datacall exists, create one. The
		// integration test prefix makes it discoverable by the
		// startup/teardown sweep regardless of how the test exits.
		t.Log("no future-deadline datacall present; using a synthesized one for this test")
		name := fmt.Sprintf("%sfuture_%d", integrationTestPrefix, time.Now().UnixNano())
		err = conn.QueryRow(ctx, `
			INSERT INTO datacalls (datacall, datecreated, deadline)
			VALUES ($1, NOW(), NOW() + INTERVAL '7 days')
			RETURNING datacallid
		`, name).Scan(&futureDataCallID)
		require.NoError(t, err)
	}

	t.Run("NotesTooLong_ReturnsErr", func(t *testing.T) {
		bigNotes := strings.Repeat("x", 2001)
		s := &Score{
			FismaSystemID:    1,
			FunctionOptionID: 1,
			DataCallID:       futureDataCallID,
			Notes:            &bigNotes,
		}
		_, err := s.Save(ctx)
		assert.ErrorIs(t, err, ErrNotesTooLong, "2001-char notes must trip the length guard")
	})

	t.Run("PastDeadline_NonAdmin_ReturnsErr", func(t *testing.T) {
		notes := "after deadline"
		s := &Score{
			FismaSystemID:    1,
			FunctionOptionID: 1,
			DataCallID:       pastDataCallID,
			Notes:            &notes,
		}
		// No user in context = treated as non-admin per validate()
		_, err := s.Save(ctx)
		assert.ErrorIs(t, err, ErrPastDeadline,
			"past-deadline save without admin must trip deadline guard")
	})

	t.Run("PastDeadline_Admin_Allowed", func(t *testing.T) {
		notes := "after deadline but admin"
		s := &Score{
			FismaSystemID:    1,
			FunctionOptionID: 1,
			DataCallID:       pastDataCallID,
			Notes:            &notes,
		}
		adminCtx := UserToContext(ctx, &User{
			UserID: "00000000-0000-4000-8000-000000000000",
			Role:   "OWNER",
		})
		// Admin bypasses the deadline guard. The save itself may fail
		// for unrelated reasons (FK violation if system 1 / option 1
		// not seeded) — but the error must NOT be ErrPastDeadline.
		_, saveErr := s.Save(adminCtx)
		if saveErr != nil {
			assert.NotErrorIs(t, saveErr, ErrPastDeadline,
				"admin save past deadline must not be blocked by the deadline guard")
		}
		// Best-effort cleanup if it did succeed
		if s.ScoreID != 0 {
			_, _ = conn.Exec(ctx, `DELETE FROM scores WHERE scoreid = $1`, s.ScoreID)
		}
	})
}

// TestCopyPreviousScoresIntegration verifies the rollover function
// carries forward scores from one datacall to the next. Creates two
// synthetic datacalls, attaches scores to the first, runs the copy,
// asserts the second now has the same set of (functionoptionid,
// fismasystemid) entries.
func TestCopyPreviousScoresIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	// Insert two datacalls explicitly ordered so copyPreviousScores'
	// "latest-1 is previous" logic finds the right one. Each row gets
	// a unique nano-precision suffix so reruns don't collide on the
	// UNIQUE(datacall) constraint, and the integrationTestPrefix makes
	// them discoverable by the sweep no matter how the test exits.
	var prevDC, newDC int32
	prevTimestamp := time.Now().Add(-2 * time.Hour)
	newTimestamp := time.Now().Add(-1 * time.Hour)
	suffix := time.Now().UnixNano()

	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, $2::timestamptz, $2::timestamptz + INTERVAL '90 days')
		RETURNING datacallid
	`, fmt.Sprintf("%sprev_%d", integrationTestPrefix, suffix), prevTimestamp).Scan(&prevDC)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, $2::timestamptz, $2::timestamptz + INTERVAL '90 days')
		RETURNING datacallid
	`, fmt.Sprintf("%snew_%d", integrationTestPrefix, suffix), newTimestamp).Scan(&newDC)
	require.NoError(t, err)

	// Pick a (fismasystemid, functionoptionid) that already references
	// a real function so the FK constraint is satisfied. Any active
	// system with at least one scored functionoption will do.
	var fismaSystemID, functionOptionID int32
	err = conn.QueryRow(ctx, `
		SELECT s.fismasystemid, s.functionoptionid
		FROM scores s LIMIT 1
	`).Scan(&fismaSystemID, &functionOptionID)
	require.NoError(t, err, "need at least one existing score row to derive a valid (system, functionoption) pair")

	// Insert a marker score under the prev datacall.
	notes := "integration test marker"
	_, err = conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes)
		VALUES ($1, $2, $3, $4)
	`, fismaSystemID, functionOptionID, prevDC, notes)
	require.NoError(t, err)

	// Before copy: the new datacall has zero scores.
	var beforeCount int
	err = conn.QueryRow(ctx,
		`SELECT COUNT(*) FROM scores WHERE datacallid = $1`, newDC,
	).Scan(&beforeCount)
	require.NoError(t, err)
	require.Equal(t, 0, beforeCount, "newDC starts with no scores")

	// Run the rollover. copyPreviousScores is unexported and accepts
	// the *latest* datacallid — it discovers the previous one via
	// findPreviousDataCall.
	copyPreviousScores(newDC)

	// After copy: the new datacall has at least the marker score.
	var afterCount int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM scores
		WHERE datacallid = $1
		  AND fismasystemid = $2
		  AND functionoptionid = $3
	`, newDC, fismaSystemID, functionOptionID).Scan(&afterCount)
	require.NoError(t, err)
	assert.Equal(t, 1, afterCount,
		"copyPreviousScores must carry the marker (system, functionoption) into the new datacall")
}

func boolPtr(b bool) *bool { return &b }

// ensureFutureDataCall returns a datacallid whose deadline is in the
// future, synthesizing one (marked with integrationTestPrefix so the
// sweep cleans it up) if none exists in seed data. Shared by the audit
// field tests below so each one does not redo the discovery dance.
//
// In the default empire seed this resolves to datacallid=3 ("Audit
// Fields Smoke Cycle", deadline 2099-12-31), which is also the cycle
// the emberfall audit-fields block writes to. That is intentional
// reuse, not a fixture collision: scores written by these integration
// tests carry their own scoreid and are cleaned up by the per-test
// defer, so they cannot leak into the emberfall HTTP-layer pass.
func ensureFutureDataCall(t *testing.T, ctx context.Context) int32 {
	t.Helper()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	var dataCallID int32
	err = conn.QueryRow(ctx, `
		SELECT datacallid FROM datacalls WHERE deadline > NOW() ORDER BY deadline ASC LIMIT 1
	`).Scan(&dataCallID)
	if err == nil {
		return dataCallID
	}

	name := fmt.Sprintf("%saudit_future_%d", integrationTestPrefix, time.Now().UnixNano())
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), NOW() + INTERVAL '7 days')
		RETURNING datacallid
	`, name).Scan(&dataCallID))
	return dataCallID
}

// anyExistingFunctionOption returns a (fismasystemid, functionoptionid)
// pair drawn from existing scores so the FK constraints hold without
// requiring those rows to be associated with any specific datacall.
func anyExistingFunctionOption(t *testing.T, ctx context.Context) (int32, int32) {
	t.Helper()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	var fismaSystemID, functionOptionID int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT s.fismasystemid, s.functionoptionid
		FROM scores s LIMIT 1
	`).Scan(&fismaSystemID, &functionOptionID),
		"need at least one existing score to derive an FK-safe (system, functionoption) pair")
	return fismaSystemID, functionOptionID
}

// TestScoreSaveStampsAuditFieldsIntegration verifies the write-side audit
// contract: Save with a user in context returns a Score whose
// LastEditedAt and LastEditedBy reflect that user. This is what the
// frontend reads off the POST/PUT response without a follow-up GET.
//
// Empire fixtures only (see [[feedback_no_real_pii_in_tests]]); never
// substitute real CMS users.
func TestScoreSaveStampsAuditFieldsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fismaSystemID, functionOptionID := anyExistingFunctionOption(t, ctx)
	dataCallID := ensureFutureDataCall(t, ctx)

	notes := "audit stamp integration test"
	s := &Score{
		FismaSystemID:    fismaSystemID,
		FunctionOptionID: functionOptionID,
		DataCallID:       dataCallID,
		Notes:            &notes,
	}

	tarkinCtx := UserToContext(ctx, &User{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "Grand.Moff@DeathStar.Empire",
		FullName: "Grand Moff Tarkin",
		Role:     "OWNER",
	})

	before := time.Now().UTC()
	saved, err := s.Save(tarkinCtx)
	require.NoError(t, err)
	require.NotNil(t, saved)
	defer func() {
		// Cleanup regardless of assertion outcome.
		_, _ = conn.Exec(ctx, `DELETE FROM scores WHERE scoreid = $1`, saved.ScoreID)
	}()

	assert.Greater(t, saved.ScoreID, int32(0), "Save returned a scoreid")
	require.NotNil(t, saved.LastEditedAt, "LastEditedAt populated on write response")
	assert.False(t, saved.LastEditedAt.Before(before),
		"LastEditedAt at or after the moment of Save")
	require.NotNil(t, saved.LastEditedBy, "LastEditedBy populated on write response")
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", saved.LastEditedBy.UserID)
	assert.Equal(t, "Grand Moff Tarkin", saved.LastEditedBy.Name)
	assert.Equal(t, "Grand.Moff@DeathStar.Empire", saved.LastEditedBy.Email)
	assert.Equal(t, "OWNER", saved.LastEditedBy.Role)
}

// TestFindScoresIncludesAuditFieldsIntegration verifies the read-side
// audit contract: after a Save through the model, a subsequent
// FindScores returns the same row with LastEditedAt + LastEditedBy
// populated via the LATERAL join on events. This is what the dashboard
// list view consumes.
//
// Pairs with TestScoreSaveStampsAuditFieldsIntegration: write-side
// stamps the response in the same call; read-side resolves it from the
// recorded event. Both must agree on the same user.
func TestFindScoresIncludesAuditFieldsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fismaSystemID, functionOptionID := anyExistingFunctionOption(t, ctx)
	dataCallID := ensureFutureDataCall(t, ctx)

	notes := "audit read integration test"
	s := &Score{
		FismaSystemID:    fismaSystemID,
		FunctionOptionID: functionOptionID,
		DataCallID:       dataCallID,
		Notes:            &notes,
	}

	piettCtx := UserToContext(ctx, &User{
		UserID:   "22222222-2222-2222-2222-222222222222",
		Email:    "Admiral.Piett@executor.empire",
		FullName: "Admiral Piett",
		Role:     "ISSO",
	})

	saved, err := s.Save(piettCtx)
	require.NoError(t, err)
	require.NotNil(t, saved)
	defer func() {
		_, _ = conn.Exec(ctx, `DELETE FROM scores WHERE scoreid = $1`, saved.ScoreID)
	}()

	// FindScores via the same filters; the saved row must be present and
	// carry audit fields resolved from the events table.
	scores, err := FindScores(ctx, FindScoresInput{
		FismaSystemID: &fismaSystemID,
		DataCallID:    &dataCallID,
	})
	require.NoError(t, err)

	var found *Score
	for _, sc := range scores {
		if sc.ScoreID == saved.ScoreID {
			found = sc
			break
		}
	}
	require.NotNil(t, found, "FindScores must return the row we just Saved")

	require.NotNil(t, found.LastEditedAt, "lateral join populated LastEditedAt")
	require.NotNil(t, found.LastEditedBy, "lateral join resolved editor")
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", found.LastEditedBy.UserID,
		"editor identity must match the user-in-context that performed the Save")
	assert.Equal(t, "Admiral Piett", found.LastEditedBy.Name)
	assert.Equal(t, "Admiral.Piett@executor.empire", found.LastEditedBy.Email)
	assert.Equal(t, "ISSO", found.LastEditedBy.Role)
}

// TestFindScoresResolvesSeededAuditFieldsIntegration covers the historical
// read path that the other audit tests do not: a score whose edit event was
// recorded by seed data (not written in-test) must still resolve LastEditedBy
// through the LATERAL join on events. This is exactly what a fresh local dev
// environment and the dashboard "Last edited <when> by <who>" footer rely on
// when nothing has been edited in the current session.
//
// The fixture is the expanded empire seed: system 1110 (Tarkin Initiative
// Superweapon R&D), scored in datacall 2 (FY2025) with one seeded "updated"
// event per score attributed to its assigned officer, Bevel Lemelisk. If the
// seed audit events go missing or the join regresses, last_edited_by silently
// returns to blank in the UI and this test fails.
//
// Empire fixtures only (see [[feedback_no_real_pii_in_tests]]); never
// substitute real CMS users.
func TestFindScoresResolvesSeededAuditFieldsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()

	// Tarkin Initiative R&D system and the FY2025 cycle, both from the seed.
	fismaSystemID := int32(1110)
	dataCallID := int32(2)

	scores, err := FindScores(ctx, FindScoresInput{
		FismaSystemID: &fismaSystemID,
		DataCallID:    &dataCallID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, scores,
		"expanded empire seed must score system 1110 in datacall 2 (FY2025)")

	// Every seeded score for this system+cycle carries an event attributed to
	// the system's assigned officer, so each row must resolve the same editor.
	for _, sc := range scores {
		require.NotNil(t, sc.LastEditedAt,
			"lateral join must populate LastEditedAt from the seeded event")
		require.NotNil(t, sc.LastEditedBy,
			"lateral join must resolve the editor from the seeded event")
		assert.Equal(t, "f0000002-0002-4002-8002-000000000002", sc.LastEditedBy.UserID)
		assert.Equal(t, "Bevel Lemelisk", sc.LastEditedBy.Name)
		assert.Equal(t, "Bevel.Lemelisk@sienar.empire", sc.LastEditedBy.Email)
		assert.Equal(t, "ISSO", sc.LastEditedBy.Role)
	}
}

// TestScoreSaveNoOpPreservesPriorEditorIntegration pins the
// audit-preservation rule: re-saving a score with identical answer
// fields must NOT overwrite the prior editor in the events trail.
//
// Product rule (per dashboard owner): "save on real change, not on
// read-through." The questionnaire UI PUTs unconditionally on every
// Next click, so without this guard a user who walks past a question
// already answered by someone else gets stamped as the new editor.
//
// Scenario:
//  1. Krennic writes notes "X" on a score, his event is the latest.
//  2. Tarkin re-saves with the same notes and same functionoption.
//  3. lookupScoreAudit must still point at Krennic; no new event row
//     must have appeared for Tarkin.
func TestScoreSaveNoOpPreservesPriorEditorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	fismaSystemID, functionOptionID := anyExistingFunctionOption(t, ctx)
	dataCallID := ensureFutureDataCall(t, ctx)

	// Step 1: Krennic creates the score with notes "X".
	notesOriginal := "krennic original answer"
	s := &Score{
		FismaSystemID:    fismaSystemID,
		FunctionOptionID: functionOptionID,
		DataCallID:       dataCallID,
		Notes:            &notesOriginal,
	}
	krennicCtx := UserToContext(ctx, &User{
		UserID:   "44444444-4444-4444-4444-444444444444",
		Email:    "Director.Krennic@scarif.empire",
		FullName: "Orson Krennic",
		Role:     "ISSO",
	})
	saved, err := s.Save(krennicCtx)
	require.NoError(t, err)
	require.NotNil(t, saved)
	scoreID := saved.ScoreID
	defer func() { _, _ = conn.Exec(ctx, `DELETE FROM scores WHERE scoreid=$1`, scoreID) }()

	// Capture the event count for this scoreid as our baseline. We do not
	// assert an absolute count here because the dev events table accretes
	// across test runs and a recycled sequence value can leave stale event
	// rows keyed on the same scoreid from an earlier session. All the
	// downstream assertions are delta-based against this baseline, which
	// is what the audit-preservation contract actually requires (a no-op
	// Save must add zero events; a real Save must add one).
	eventsBefore := countScoreEvents(t, ctx, conn, scoreID)
	require.GreaterOrEqual(t, eventsBefore, 1, "Krennic's initial Save should record at least one event for this scoreid")

	// Step 2: Tarkin re-saves with identical fields. The FE does this on
	// every Next click; the BE must treat it as a no-op.
	tarkinCtx := UserToContext(ctx, &User{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "Grand.Moff@DeathStar.Empire",
		FullName: "Grand Moff Tarkin",
		Role:     "OWNER",
	})
	resave := &Score{
		ScoreID:          scoreID,
		FismaSystemID:    fismaSystemID,
		FunctionOptionID: functionOptionID,
		DataCallID:       dataCallID,
		Notes:            &notesOriginal,
	}
	tarkinResult, err := resave.Save(tarkinCtx)
	require.NoError(t, err)
	require.NotNil(t, tarkinResult)

	// Step 3: events count unchanged; lookupScoreAudit still points at
	// Krennic.
	eventsAfter := countScoreEvents(t, ctx, conn, scoreID)
	assert.Equal(t, eventsBefore, eventsAfter,
		"no-op Save must NOT insert a new event row (Tarkin's read-through PUT must not overwrite history)")

	require.NotNil(t, tarkinResult.LastEditedBy,
		"no-op Save response should still carry the canonical audit (Krennic), not nil")
	assert.Equal(t, "44444444-4444-4444-4444-444444444444",
		tarkinResult.LastEditedBy.UserID,
		"no-op Save response must report Krennic as the editor, not Tarkin who issued the PUT")
	assert.Equal(t, "ISSO", tarkinResult.LastEditedBy.Role)

	// Step 4: A real change DOES record a new event. Confirms the no-op
	// guard is not over-broad.
	notesChanged := "tarkin actually edited"
	realChange := &Score{
		ScoreID:          scoreID,
		FismaSystemID:    fismaSystemID,
		FunctionOptionID: functionOptionID,
		DataCallID:       dataCallID,
		Notes:            &notesChanged,
	}
	realResult, err := realChange.Save(tarkinCtx)
	require.NoError(t, err)
	require.NotNil(t, realResult)

	eventsAfterRealChange := countScoreEvents(t, ctx, conn, scoreID)
	assert.Equal(t, eventsBefore+1, eventsAfterRealChange,
		"genuine notes change must record a new event row")
	require.NotNil(t, realResult.LastEditedBy)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111",
		realResult.LastEditedBy.UserID,
		"after a real change, the editor must be the user who made it")
}

// countScoreEvents is a small helper used by the no-op preservation test
// to assert that read-through Saves do not append event rows.
func countScoreEvents(t *testing.T, ctx context.Context, conn *pgxpool.Conn, scoreID int32) int {
	t.Helper()
	var n int
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM events
		WHERE resource='public.scores'
		  AND (payload->>'scoreid')::int = $1
	`, scoreID).Scan(&n))
	return n
}

// TestFindScoresISSOScopeRetainsAuditFieldsIntegration verifies that the
// ISSO scope predicate (users_fismasystems join via input.UserID) still
// produces populated audit fields. Regression target: a future
// refactor that drops the LATERAL join when the users_fismasystems join
// is present would silently strip last-edited info from the ISSO list
// view -- the primary consumer of #310.
//
// Uses Director Krennic (assigned to fismasystemid 1003 in
// _test_data_empire.sql) as the scoped ISSO; saves a score as Krennic
// to confirm the lateral join resolves to him from his own write.
func TestFindScoresISSOScopeRetainsAuditFieldsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	const krennicUUID = "44444444-4444-4444-4444-444444444444"
	const krennicFisma = int32(1003)

	// Pick any functionoption that exists; FK to functionoptions is what
	// matters. The functionoptionid space is dense from seed data, so the
	// first row will do.
	var functionOptionID int32
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT functionoptionid FROM functionoptions LIMIT 1`).Scan(&functionOptionID))

	dataCallID := ensureFutureDataCall(t, ctx)

	notes := "ISSO scope audit retention test"
	s := &Score{
		FismaSystemID:    krennicFisma,
		FunctionOptionID: functionOptionID,
		DataCallID:       dataCallID,
		Notes:            &notes,
	}
	krennicCtx := UserToContext(ctx, &User{
		UserID:   krennicUUID,
		Email:    "Director.Krennic@scarif.empire",
		FullName: "Orson Krennic",
		Role:     "ISSO",
	})
	saved, err := s.Save(krennicCtx)
	require.NoError(t, err)
	defer func() {
		_, _ = conn.Exec(ctx, `DELETE FROM scores WHERE scoreid = $1`, saved.ScoreID)
	}()

	// Same scope the controller applies for an ISSO: UserID set, no
	// FismaSystemID filter -- the users_fismasystems join scopes to
	// Krennic's assigned systems.
	krennicUID := krennicUUID
	scores, err := FindScores(ctx, FindScoresInput{
		UserID: &krennicUID,
	})
	require.NoError(t, err)

	var found *Score
	for _, sc := range scores {
		if sc.ScoreID == saved.ScoreID {
			found = sc
			break
		}
		// Every row returned must be a system Krennic is assigned to;
		// the seeded assignment is fismasystemid=1003 only.
		assert.Equal(t, krennicFisma, sc.FismaSystemID,
			"ISSO scope leaked: row for unassigned system in Krennic's list")
	}
	require.NotNil(t, found, "Krennic's just-saved row must appear in his scoped list")
	require.NotNil(t, found.LastEditedBy, "ISSO scope must not strip the audit join")
	assert.Equal(t, krennicUUID, found.LastEditedBy.UserID)
	assert.Equal(t, "ISSO", found.LastEditedBy.Role)
}
