package model

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assignedopdivids is an always-array on the wire (ztmf#346). The nil-vs-empty
// distinction is invisible to Go's own consumers - len() and range treat them
// alike - so it is pinned on both the scanned slice and the marshalled JSON.

const noGrantUserEmail = "No.Grants.User@empire.test"

// Resolved rather than hardcoded so the test cannot pass vacuously against a
// fixture where every user happens to be granted.
func grantlessUserID(t *testing.T, ctx context.Context) string {
	t.Helper()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	var userID string
	err = conn.QueryRow(ctx, `
		SELECT u.userid FROM public.users u
		WHERE NOT EXISTS (SELECT 1 FROM public.users_opdivs uo WHERE uo.userid = u.userid)
		LIMIT 1
	`).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		t.Skip("fixture has no user without OpDiv grants; nothing to assert here")
	}
	require.NoError(t, err)
	return userID
}

// Both halves of the contract: a non-nil empty slice, and a literal [] on the wire.
func assertEmptyArray(t *testing.T, u *User, where string) {
	t.Helper()
	assert.NotNil(t, u.AssignedOpDivIDs, "%s: a grantless user must scan an empty slice, not nil", where)
	assert.Empty(t, u.AssignedOpDivIDs, "%s: a grantless user must have no grants", where)

	b, err := json.Marshal(u)
	require.NoError(t, err)
	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &wire))
	raw, ok := wire["assignedopdivids"]
	require.True(t, ok, "%s: assignedopdivids must be present in the response", where)
	assert.JSONEq(t, `[]`, string(raw), "%s: must serialize as [], never null", where)
}

func TestAssignedOpDivIDsEmptyArrayIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()
	userID := grantlessUserID(t, ctx)

	t.Run("FindUsers", func(t *testing.T) {
		users, err := FindUsers(ctx, &FindUsersInput{})
		require.NoError(t, err)
		require.NotEmpty(t, users)

		var subject *User
		for _, u := range users {
			if u.UserID == userID {
				subject = u
				break
			}
		}
		require.NotNil(t, subject, "the grantless user should appear in the list")
		assertEmptyArray(t, subject, "FindUsers")
	})

	t.Run("FindUserByID", func(t *testing.T) {
		u, err := FindUserByID(ctx, userID)
		require.NoError(t, err)
		require.NotNil(t, u)
		assertEmptyArray(t, u, "findUser")
	})

	// A COALESCE that flattened everyone to empty would satisfy both assertions
	// above, so check every user against the actual junction-table counts.
	t.Run("GrantsAreNotFlattened", func(t *testing.T) {
		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn.Release()

		rows, err := conn.Query(ctx, `
			SELECT userid, count(*) FROM public.users_opdivs GROUP BY userid
		`)
		require.NoError(t, err)
		want := map[string]int{}
		for rows.Next() {
			var id string
			var n int
			require.NoError(t, rows.Scan(&id, &n))
			want[id] = n
		}
		require.NoError(t, rows.Err())
		require.NotEmpty(t, want, "fixture must grant at least one OpDiv")

		// Deleted=false is the default filter, so check both halves of the
		// fixture to cover every granted user.
		var matched int
		for _, deleted := range []bool{false, true} {
			users, err := FindUsers(ctx, &FindUsersInput{Deleted: deleted})
			require.NoError(t, err)
			for _, u := range users {
				assert.NotNil(t, u.AssignedOpDivIDs, "%s: never nil, grants or not", u.UserID)
				assert.Len(t, u.AssignedOpDivIDs, want[u.UserID],
					"%s: list must report the user's actual grant count", u.UserID)
				if want[u.UserID] > 0 {
					matched++
				}
			}
		}
		assert.Equal(t, len(want), matched, "every granted user should have been checked")
	})
}

// The half the acceptance criteria did not name: Save and RestoreUser omitted
// assignedopdivids from RETURNING entirely, so lax scanning returned null.
func TestAssignedOpDivIDsWritePathsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()

	hardDeleteUserByEmail(t, noGrantUserEmail)
	t.Cleanup(func() { hardDeleteUserByEmail(t, noGrantUserEmail) })

	u := &User{Email: noGrantUserEmail, FullName: "No Grants User", Role: "ISSO"}
	created, err := u.Save(ctx)
	require.NoError(t, err)
	require.NotNil(t, created)
	assertEmptyArray(t, created, "Save (create)")

	created.FullName = "No Grants User Renamed"
	updated, err := created.Save(ctx)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assertEmptyArray(t, updated, "Save (update)")

	require.NoError(t, DeleteUser(ctx, created.UserID))
	restored, err := RestoreUser(ctx, created.UserID)
	require.NoError(t, err)
	require.NotNil(t, restored)
	assertEmptyArray(t, restored, "RestoreUser")

	// Same user, now granted: the write paths must report it, not a blanket
	// empty array. Uses a user this test created so the id is a v4 UUID
	// (FindUserByID rejects the fixture's 3333-style ids).
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()
	var opdivID int32
	require.NoError(t, conn.QueryRow(ctx, "SELECT opdiv_id FROM opdivs ORDER BY opdiv_id LIMIT 1").Scan(&opdivID))
	_, err = conn.Exec(ctx,
		"INSERT INTO users_opdivs (userid, opdiv_id, granted_by) VALUES ($1, $2, $1)",
		created.UserID, opdivID)
	require.NoError(t, err)

	detail, err := FindUserByID(ctx, created.UserID)
	require.NoError(t, err)
	require.Len(t, detail.AssignedOpDivIDs, 1, "findUser must report the grant")
	require.NotNil(t, detail.AssignedOpDivIDs[0])
	assert.Equal(t, opdivID, *detail.AssignedOpDivIDs[0])

	granted, err := detail.Save(ctx)
	require.NoError(t, err)
	assert.Len(t, granted.AssignedOpDivIDs, 1, "Save (update) must report the grant")
}
