package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A PUT must never move a score row to another system or another data call.
// Both columns decide who may write the row and which deadline applies to it,
// so a request body that rewrites them turns an answer edit into a reparenting
// primitive (ztmf-misc#263). Save omits both from the UPDATE's SET list; this
// pins that at the model layer, independent of whatever the controller passes.
func TestScoreSaveCannotReparentIntegration(t *testing.T) {
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

	// A second system and a second data call to attempt the move toward.
	var otherSystemID int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT fismasystemid FROM fismasystems
		WHERE fismasystemid <> $1 AND COALESCE(decommissioned,false) = false
		LIMIT 1
	`, fismaSystemID).Scan(&otherSystemID), "need a second system to attempt a reparent")
	require.NotEqual(t, fismaSystemID, otherSystemID)

	var otherDataCallID int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT datacallid FROM datacalls WHERE datacallid <> $1 LIMIT 1
	`, dataCallID).Scan(&otherDataCallID), "need a second data call to attempt a move")
	require.NotEqual(t, dataCallID, otherDataCallID)

	owner := UserToContext(ctx, &User{
		UserID:   "11111111-1111-1111-1111-111111111111",
		Email:    "Grand.Moff@DeathStar.Empire",
		FullName: "Grand Moff Tarkin",
		Role:     "OWNER",
	})

	notes := "original answer"
	created, err := (&Score{
		FismaSystemID:    fismaSystemID,
		FunctionOptionID: functionOptionID,
		DataCallID:       dataCallID,
		Notes:            &notes,
	}).Save(owner)
	require.NoError(t, err)
	require.NotNil(t, created)
	scoreID := created.ScoreID
	defer func() { _, _ = conn.Exec(ctx, `DELETE FROM scores WHERE scoreid=$1`, scoreID) }()

	// The attack shape: same scoreid, but the receiver names a different
	// system and a different cycle, alongside a real answer change so the
	// write is not skipped as a no-op.
	//
	// The UPDATE is bound to (scoreid, fismasystemid, datacallid), so a
	// receiver disagreeing with the stored row matches nothing and fails
	// rather than silently applying the answer half of the write. That is the
	// model-layer defense: it holds even if a caller forgets to pin the two
	// fields from storage, which is the mistake that made ztmf-misc#263
	// reachable in the first place.
	moved := "answer rewritten by the reparent attempt"
	_, err = (&Score{
		ScoreID:          scoreID,
		FismaSystemID:    otherSystemID,
		DataCallID:       otherDataCallID,
		FunctionOptionID: functionOptionID,
		Notes:            &moved,
	}).Save(owner)
	require.Error(t, err,
		"a Save whose receiver disagrees with the stored row must fail, not partially apply")

	var gotSystem, gotCall int32
	var gotNotes *string
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT fismasystemid, datacallid, notes FROM scores WHERE scoreid=$1
	`, scoreID).Scan(&gotSystem, &gotCall, &gotNotes))

	assert.Equal(t, fismaSystemID, gotSystem,
		"the row must not move to the system named in the request (ztmf-misc#263)")
	assert.Equal(t, dataCallID, gotCall,
		"the row must not move to the data call named in the request (ztmf-misc#263)")
	require.NotNil(t, gotNotes)
	assert.Equal(t, notes, *gotNotes,
		"a refused reparent must leave the original answer intact, not apply the new notes")

	// The legitimate shape - same row, correct parents, changed answer - still
	// works. Without this the test above would pass against a Save that simply
	// never updates anything.
	legit := "answer updated legitimately"
	_, err = (&Score{
		ScoreID:          scoreID,
		FismaSystemID:    fismaSystemID,
		DataCallID:       dataCallID,
		FunctionOptionID: functionOptionID,
		Notes:            &legit,
	}).Save(owner)
	require.NoError(t, err)

	require.NoError(t, conn.QueryRow(ctx, `SELECT notes FROM scores WHERE scoreid=$1`, scoreID).Scan(&gotNotes))
	require.NotNil(t, gotNotes)
	assert.Equal(t, legit, *gotNotes, "a correctly-parented save must still apply")
}
