package model

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindScoreProgressIntegration pins the central semantic of ztmf#299
// against the real SQL: an answer pre-populated by copyPreviousScores does
// NOT count as updated (the copy records no events), and the same answer
// counts the moment a user actually saves it (the write path records an
// event). Unit tests over the generated SQL cannot see this - it depends on
// the interplay between the copy path, the event trigger in queryRow, and
// the INNER lateral in the progress query.
//
// Requires DB_* env vars pointing at a seeded ZTMF database (the dev compose
// stack). Skipped under `go test -short`.
func TestFindScoreProgressIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// Two synthetic datacalls ordered so copyPreviousScores' "latest-1 is
	// previous" logic finds the right one. Prefixed names make them (and
	// their scores, via FK cascade) discoverable by the purge sweep no
	// matter how the test exits.
	var prevDC, newDC int32
	prevTimestamp := time.Now().Add(-2 * time.Hour)
	newTimestamp := time.Now().Add(-1 * time.Hour)
	suffix := time.Now().UnixNano()

	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, $2::timestamptz, $2::timestamptz + INTERVAL '90 days')
		RETURNING datacallid
	`, fmt.Sprintf("%sprogress_prev_%d", integrationTestPrefix, suffix), prevTimestamp).Scan(&prevDC)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, $2::timestamptz, $2::timestamptz + INTERVAL '90 days')
		RETURNING datacallid
	`, fmt.Sprintf("%sprogress_new_%d", integrationTestPrefix, suffix), newTimestamp).Scan(&newDC)
	require.NoError(t, err)

	// Borrow a valid (system, functionoption) pair from seeded data so FK
	// constraints hold.
	// The system must be active: FindScoreProgress excludes decommissioned
	// systems, so borrowing one would make the single-system lookup empty.
	var fismaSystemID, functionOptionID int32
	err = conn.QueryRow(ctx, `
		SELECT s.fismasystemid, s.functionoptionid
		FROM scores s
		INNER JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		WHERE fs.decommissioned = FALSE
		LIMIT 1
	`).Scan(&fismaSystemID, &functionOptionID)
	require.NoError(t, err, "need at least one score row on an active system to derive a valid (system, functionoption) pair")

	// Seed one answer in the previous cycle via raw SQL (no events, same as
	// historical data), then roll it into the new cycle the way datacall
	// creation does.
	notes := "progress integration marker"
	_, err = conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes)
		VALUES ($1, $2, $3, $4)
	`, fismaSystemID, functionOptionID, prevDC, notes)
	require.NoError(t, err)

	copyPreviousScores(newDC)

	findForSystem := func() *ScoreProgress {
		t.Helper()
		rows, err := FindScoreProgress(ctx, FindScoreProgressInput{
			DataCallID:    &newDC,
			FismaSystemID: &fismaSystemID,
		})
		require.NoError(t, err)
		require.Len(t, rows, 1, "single-system filter must return exactly that system")
		require.Equal(t, fismaSystemID, rows[0].FismaSystemID)
		return rows[0]
	}

	// Phase 1: the answer exists in the new cycle (copied), but nothing has
	// been touched. Progress must read zero - this is the "has a row but was
	// not updated" case the naive row count gets wrong.
	before := findForSystem()
	assert.Equal(t, int32(0), before.QuestionsUpdated,
		"pre-populated answers must not count as updated")
	assert.False(t, before.UpdatedSinceStart)
	assert.Nil(t, before.LastUpdatedAt)
	assert.GreaterOrEqual(t, before.QuestionsExpected, int32(0),
		"expected count resolves through the environment mapping")

	// Phase 2: a user saves the copied answer through the normal write path,
	// which records an edit event. Progress must now count it.
	var copiedScoreID int32
	err = conn.QueryRow(ctx, `
		SELECT scoreid FROM scores
		WHERE datacallid = $1 AND fismasystemid = $2 AND functionoptionid = $3
	`, newDC, fismaSystemID, functionOptionID).Scan(&copiedScoreID)
	require.NoError(t, err, "the copied row must exist in the new datacall")

	// The editor must be a real seeded user: recordEvent writes
	// events.userid with an FK to users and silently swallows the insert
	// error on violation, so a fabricated UUID would make the edit
	// invisible to the progress query and this test would fail for the
	// wrong reason.
	var editorID, editorRole string
	err = conn.QueryRow(ctx, `
		SELECT userid, role FROM users ORDER BY (role = 'OWNER') DESC LIMIT 1
	`).Scan(&editorID, &editorRole)
	require.NoError(t, err, "need at least one seeded user to attribute the edit to")

	editedNotes := "edited this cycle"
	edited := &Score{
		ScoreID:          copiedScoreID,
		FismaSystemID:    fismaSystemID,
		FunctionOptionID: functionOptionID,
		DataCallID:       newDC,
		Notes:            &editedNotes,
	}
	editorCtx := UserToContext(ctx, &User{
		UserID: editorID,
		Role:   editorRole,
	})
	_, err = edited.Save(editorCtx)
	require.NoError(t, err, "editing the copied answer through Save must succeed")

	after := findForSystem()
	assert.Equal(t, int32(1), after.QuestionsUpdated,
		"a genuinely edited answer must count as updated")
	assert.True(t, after.UpdatedSinceStart)
	if assert.NotNil(t, after.LastUpdatedAt, "last update timestamp must surface from the edit event") {
		assert.WithinDuration(t, time.Now(), *after.LastUpdatedAt, 5*time.Minute)
	}
}
