package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindScoresSystemScopeIntegration pins that FindScores honors the
// assigned-systems scope: a restricted, empty scope returns no rows (not every
// row), and a restricted list returns only those systems. ListScores scopes by
// UserID today, so this guards any caller that scopes with ScopeToSystems
// instead.
func TestFindScoresSystemScopeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	// Two scored systems in one data call, so the narrowed result has
	// something to leave out.
	var dataCallID, keep, drop int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT datacallid, MIN(fismasystemid), MAX(fismasystemid) FROM scores
		GROUP BY datacallid
		HAVING COUNT(DISTINCT fismasystemid) >= 2
		ORDER BY datacallid
		LIMIT 1
	`).Scan(&dataCallID, &keep, &drop), "seed must have a data call scored for two systems")
	require.NotEqual(t, keep, drop)

	t.Run("UnrestrictedReturnsAllSystems", func(t *testing.T) {
		scores, err := FindScores(ctx, FindScoresInput{DataCallID: &dataCallID})
		require.NoError(t, err)

		seen := map[int32]bool{}
		for _, s := range scores {
			seen[s.FismaSystemID] = true
		}
		assert.True(t, seen[keep] && seen[drop], "control: both systems have rows when unrestricted")
	})

	for name, ids := range map[string][]*int32{"Nil": nil, "Empty": {}} {
		t.Run("RestrictedEmpty"+name+"ReturnsNothing", func(t *testing.T) {
			in := FindScoresInput{DataCallID: &dataCallID}
			in.ScopeToSystems(ids)

			scores, err := FindScores(ctx, in)
			require.NoError(t, err)
			assert.Empty(t, scores, "an empty restricted scope must match no rows")
		})
	}

	t.Run("RestrictedEmptyWithRequestedSystemReturnsNothing", func(t *testing.T) {
		in := FindScoresInput{DataCallID: &dataCallID, FismaSystemID: &keep}
		in.ScopeToSystems(nil)

		scores, err := FindScores(ctx, in)
		require.NoError(t, err)
		assert.Empty(t, scores, "a requested system must not stand in for the empty scope")
	})

	t.Run("RestrictedListReturnsOnlyThoseSystems", func(t *testing.T) {
		in := FindScoresInput{DataCallID: &dataCallID}
		in.ScopeToSystems([]*int32{&keep})

		scores, err := FindScores(ctx, in)
		require.NoError(t, err)
		require.NotEmpty(t, scores)
		for _, s := range scores {
			assert.Equal(t, keep, s.FismaSystemID, "only the scoped system may be returned")
		}
	})
}
