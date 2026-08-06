package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FindFunctionOptions must return answer choices in maturity order (score
// ascending: traditional, initial, advanced, optimal - the CISA ZTMM stage
// sequence). Before the explicit ORDER BY, the query returned heap order,
// which happened to match for years until the 2026-08-02 description scrub
// relocated updated tuples and scrambled ~50 visible questions
// (ztmf-misc#279).
//
// The test reproduces that trigger rather than trusting fixture luck: it
// UPDATEs one option's description, which physically relocates the row in
// Postgres, then asserts the order still comes back by score. Without the
// ORDER BY this fails; with it, heap layout is irrelevant.
func TestFindFunctionOptionsOrderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)

	// Pick a real function with a full option set from the seed.
	var functionID int32
	var origDesc string
	var relocatedID int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT fo.functionid, fo.functionoptionid, fo.description
		FROM functionoptions fo
		GROUP BY fo.functionid, fo.functionoptionid, fo.description, fo.score
		HAVING (SELECT count(*) FROM functionoptions x WHERE x.functionid = fo.functionid) = 4
		ORDER BY fo.functionid, fo.score DESC LIMIT 1
	`).Scan(&functionID, &relocatedID, &origDesc))

	// Registered before the mutation; restores the description (a second
	// relocation, but content-identical). Not deferred: deferred calls run
	// BEFORE t.Cleanup and would release the connection out from under this.
	t.Cleanup(func() {
		_, _ = conn.Exec(ctx,
			`UPDATE functionoptions SET description=$1 WHERE functionoptionid=$2`,
			origDesc, relocatedID)
		conn.Release()
	})

	// The trigger: an UPDATE relocates the tuple to a new heap page, exactly
	// what the production description scrub did.
	_, err = conn.Exec(ctx,
		`UPDATE functionoptions SET description=description||' ' WHERE functionoptionid=$1`,
		relocatedID)
	require.NoError(t, err)

	opts, err := FindFunctionOptions(ctx, FindFunctionOptionsInput{FunctionID: &functionID})
	require.NoError(t, err)
	require.Len(t, opts, 4)

	for i, o := range opts {
		assert.Equal(t, i+1, int(o.Score),
			"choices must render traditional(1) -> optimal(4) regardless of heap layout")
	}
}
