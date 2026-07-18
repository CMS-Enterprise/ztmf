package model

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
)

// Empire-seed ISSO with a v4 userid. The v4 shape matters: both the model's
// validate() and FindUserByID gate on the version-4-strict isValidUUID, so the
// all-digits placeholder users (11111111-..., 22222222-...) are rejected before
// any DB work and cannot exercise this path.
const upsertTestUserID = "aa000088-8888-4888-8888-888888888888"

// Not assigned to upsertTestUserID in the empire seed; the test also clears the
// pair up front so a leftover row from an interrupted run can't skew it.
const upsertTestSystemID int32 = 1003

func countUserFismaSystemRows(t *testing.T, userID string, systemID int32) int {
	t.Helper()
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Release()
	var n int
	err = conn.QueryRow(context.Background(),
		"SELECT count(*) FROM users_fismasystems WHERE userid=$1 AND fismasystemid=$2",
		userID, systemID).Scan(&n)
	require.NoError(t, err)
	return n
}

// Regression for #429: Save used ON CONFLICT DO NOTHING RETURNING, which
// returns zero rows on conflict, so re-assigning an existing (userid,
// fismasystemid) surfaced pgx.ErrNoRows as ErrNoData (a 404 at the API layer)
// instead of being idempotent.
func TestUserFismaSystemSaveIsIdempotentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	uf := &UserFismaSystem{UserID: upsertTestUserID, FismaSystemID: upsertTestSystemID}

	// Belt-and-suspenders: clear the pair so an interrupted prior run can't
	// turn the "first" save into a duplicate. ErrNoData just means it was
	// already absent.
	if err := uf.Delete(ctx); err != nil && !errors.Is(err, ErrNoData) {
		t.Fatalf("pre-test cleanup failed: %v", err)
	}
	t.Cleanup(func() {
		if err := uf.Delete(context.Background()); err != nil && !errors.Is(err, ErrNoData) {
			t.Errorf("post-test cleanup failed: %v", err)
		}
	})

	// First save inserts and returns the row.
	first, err := uf.Save(ctx)
	require.NoError(t, err, "fresh assignment must succeed")
	require.NotNil(t, first)
	assert.Equal(t, upsertTestUserID, first.UserID)
	assert.Equal(t, upsertTestSystemID, first.FismaSystemID)
	assert.Equal(t, 1, countUserFismaSystemRows(t, upsertTestUserID, upsertTestSystemID))

	// Duplicate save is a no-op that still returns the existing row.
	second, err := uf.Save(ctx)
	require.NoError(t, err, "duplicate assignment must be idempotent, not an error (#429)")
	require.NotNil(t, second)
	assert.Equal(t, upsertTestUserID, second.UserID)
	assert.Equal(t, upsertTestSystemID, second.FismaSystemID)

	// Still exactly one row — the upsert must not duplicate.
	assert.Equal(t, 1, countUserFismaSystemRows(t, upsertTestUserID, upsertTestSystemID))
}
