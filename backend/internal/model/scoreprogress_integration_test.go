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

	// copyPreviousScores resolves "previous" via findPreviousDataCall, which
	// returns the datacall with the furthest deadline other than the target (it
	// is only ever called on the newest cycle, so "latest, excluding me" is the
	// real previous). The seed carries a far-future cycle (the 2099 "Audit Fields
	// Smoke Cycle", datacallid 5), so the synthetic cycle must sit ABOVE every
	// existing deadline: newDC must be the global latest and prevDC the next one
	// down, or findPreviousDataCall(newDC) rolls a seed datacall forward instead
	// of our marker. Anchor both deadlines above the current max. Prefixed names
	// make them (and their scores, via FK cascade) discoverable by the purge
	// sweep no matter how the test exits.
	var prevDC, newDC int32
	var maxDeadline time.Time
	err = conn.QueryRow(ctx, `SELECT COALESCE(MAX(deadline), NOW()) FROM datacalls`).Scan(&maxDeadline)
	require.NoError(t, err)
	prevDeadline := maxDeadline.Add(24 * time.Hour)
	newDeadline := maxDeadline.Add(48 * time.Hour)
	suffix := time.Now().UnixNano()

	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, $2::timestamptz, $3::timestamptz)
		RETURNING datacallid
	`, fmt.Sprintf("%sprogress_prev_%d", integrationTestPrefix, suffix), time.Now().Add(-2*time.Hour), prevDeadline).Scan(&prevDC)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, $2::timestamptz, $3::timestamptz)
		RETURNING datacallid
	`, fmt.Sprintf("%sprogress_new_%d", integrationTestPrefix, suffix), time.Now().Add(-1*time.Hour), newDeadline).Scan(&newDC)
	require.NoError(t, err)

	// Borrow a valid (system, functionoption) pair from seeded data so FK
	// constraints hold. The system must be active (FindScoreProgress excludes
	// decommissioned systems) and the answered function must be applicable to
	// the system's current environment and carry a question - otherwise the
	// progress query's applicability filter would exclude the edited answer
	// and the "counts as updated" assertion below would fail for the wrong
	// reason.
	var fismaSystemID, functionOptionID int32
	err = conn.QueryRow(ctx, `
		SELECT s.fismasystemid, s.functionoptionid
		FROM scores s
		INNER JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		INNER JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
		INNER JOIN functions f ON f.functionid = fo.functionid
		INNER JOIN datacenterenvironments dce
			ON dce.datacenterenvironment = fs.datacenterenvironment
			AND dce.scoring_key = f.datacenterenvironment
		INNER JOIN questions q ON q.questionid = f.questionid
		WHERE fs.decommissioned = FALSE
		LIMIT 1
	`).Scan(&fismaSystemID, &functionOptionID)
	require.NoError(t, err, "need one seeded score on an active system whose function is applicable to its environment")

	// Seed one answer in the previous cycle via raw SQL (no events, same as
	// historical data), then roll it into the new cycle the way datacall
	// creation does.
	notes := "progress integration marker"
	_, err = conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, functionoptionid, datacallid, notes)
		VALUES ($1, $2, $3, $4)
	`, fismaSystemID, functionOptionID, prevDC, notes)
	require.NoError(t, err)

	if _, err := copyPreviousScores(ctx, newDC); err != nil {
		t.Fatalf("copyPreviousScores: %v", err)
	}

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
	assert.LessOrEqual(t, after.QuestionsUpdated, after.QuestionsExpected,
		"updated can never exceed the applicable-question denominator")
	if assert.NotNil(t, after.LastUpdatedAt, "last update timestamp must surface from the edit event") {
		assert.WithinDuration(t, time.Now(), *after.LastUpdatedAt, 5*time.Minute)
	}
}

// TestFindScoreProgressExcludesInapplicableAnswers pins the fix for the
// >100% edge case (PR #404 review): an edited answer for a function that is
// NOT applicable to the system's current environment must not count toward
// questionsupdated, so the numerator can never exceed the applicable-question
// denominator. This is the carried-over-answer-after-an-environment-change
// scenario - copyPreviousScores copies functionoptionid verbatim, so a system
// whose environment changed between cycles can hold answers for functions that
// dropped out of its questionnaire.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindScoreProgressExcludesInapplicableAnswers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// Find an active system that (a) has at least one applicable function of
	// its own (so questionsexpected > 0) and (b) has a functionoption for a
	// function that is NOT applicable to its environment - the cross-environment
	// answer we want to prove is excluded.
	var fismaSystemID, inapplicableFOID int32
	err = conn.QueryRow(ctx, `
		SELECT fs.fismasystemid, fo.functionoptionid
		FROM fismasystems fs
		INNER JOIN functionoptions fo ON TRUE
		INNER JOIN functions f ON f.functionid = fo.functionid
		INNER JOIN questions q ON q.questionid = f.questionid
		WHERE fs.decommissioned = FALSE
		  AND fs.datacenterenvironment IS NOT NULL
		  AND f.datacenterenvironment NOT IN (
		      SELECT dce.scoring_key FROM datacenterenvironments dce
		      WHERE dce.datacenterenvironment = fs.datacenterenvironment
		  )
		  AND EXISTS (
		      SELECT 1
		      FROM datacenterenvironments dce2
		      INNER JOIN functions f2 ON f2.datacenterenvironment = dce2.scoring_key
		      INNER JOIN questions q2 ON q2.questionid = f2.questionid
		      WHERE dce2.datacenterenvironment = fs.datacenterenvironment
		  )
		LIMIT 1
	`).Scan(&fismaSystemID, &inapplicableFOID)
	if err != nil {
		t.Skip("seed has no active system with both applicable functions and a cross-environment functionoption; cannot exercise the inapplicable-answer path")
	}

	var dc int32
	suffix := time.Now().UnixNano()
	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), NOW() + INTERVAL '90 days')
		RETURNING datacallid
	`, fmt.Sprintf("%sinapplicable_%d", integrationTestPrefix, suffix)).Scan(&dc)
	require.NoError(t, err)

	// A real seeded editor - recordEvent's userid FK-references users and
	// swallows a violation silently, which would make the edit invisible.
	var editorID, editorRole string
	err = conn.QueryRow(ctx, `
		SELECT userid, role FROM users ORDER BY (role = 'OWNER') DESC LIMIT 1
	`).Scan(&editorID, &editorRole)
	require.NoError(t, err)
	editorCtx := UserToContext(ctx, &User{UserID: editorID, Role: editorRole})

	// Genuinely edit an answer for the inapplicable function (records an event).
	notes := "cross-environment answer"
	saved, err := (&Score{
		FismaSystemID:    fismaSystemID,
		FunctionOptionID: inapplicableFOID,
		DataCallID:       dc,
		Notes:            &notes,
	}).Save(editorCtx)
	require.NoError(t, err, "saving the cross-environment answer must succeed")
	require.NotZero(t, saved.ScoreID)

	rows, err := FindScoreProgress(ctx, FindScoreProgressInput{
		DataCallID:    &dc,
		FismaSystemID: &fismaSystemID,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	p := rows[0]

	assert.Greater(t, p.QuestionsExpected, int32(0),
		"the system has applicable functions of its own")
	assert.Equal(t, int32(0), p.QuestionsUpdated,
		"an edited answer for a non-applicable function must not count as updated")
	assert.False(t, p.UpdatedSinceStart,
		"updating only a non-applicable answer is not questionnaire progress")
	assert.LessOrEqual(t, p.QuestionsUpdated, p.QuestionsExpected,
		"updated can never exceed the applicable-question denominator")
}
