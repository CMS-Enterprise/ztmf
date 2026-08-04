package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RecordLogin is what makes LastSeen mean "last signed in" rather than "last
// changed something". Before it, activity was only visible for users who wrote
// data, so an account that signed in and only read was indistinguishable from
// one that had never signed in - the exact pair you need to separate to find a
// stale privileged account.
func TestRecordLoginIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	// Registered before the skip below so the connection is returned on every
	// exit path. Not a deferred release: deferred calls run BEFORE t.Cleanup,
	// which would pull the connection out from under the DELETE registered
	// later.
	var userID string
	t.Cleanup(func() {
		if userID != "" {
			_, _ = conn.Exec(ctx,
				`DELETE FROM public.events WHERE userid=$1 AND resource='session'`, userID)
		}
		conn.Release()
	})

	// A user who has recorded nothing, so the assertions below cannot be
	// satisfied by pre-existing activity.
	err = conn.QueryRow(ctx, `
		SELECT u.userid FROM public.users u
		WHERE NOT COALESCE(u.deleted,false)
		  AND NOT EXISTS (SELECT 1 FROM public.events e WHERE e.userid = u.userid)
		LIMIT 1
	`).Scan(&userID)
	if err != nil {
		t.Skip("fixture has no user without events")
	}

	before, err := FindUsers(ctx, &FindUsersInput{})
	require.NoError(t, err)
	for _, u := range before {
		if u.UserID == userID {
			require.Nil(t, u.LastSeen, "precondition: this user must start with no activity")
		}
	}

	require.NoError(t, RecordLogin(ctx, userID))

	t.Run("WritesASessionEvent", func(t *testing.T) {
		var action, resource, payloadUser string
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT action, resource, payload->>'userid'
			FROM public.events WHERE userid=$1 AND resource='session'
			ORDER BY createdat DESC LIMIT 1
		`, userID).Scan(&action, &resource, &payloadUser))
		assert.Equal(t, "created", action)
		assert.Equal(t, "session", resource)
		assert.Equal(t, userID, payloadUser, "payload should identify the user who signed in")
	})

	t.Run("SurfacesAsLastSeenWithoutAnyWrite", func(t *testing.T) {
		// The point of the change: signing in alone now moves last_seen. This
		// user still has not created or updated a single record.
		after, err := FindUsers(ctx, &FindUsersInput{})
		require.NoError(t, err)
		var found *User
		for _, u := range after {
			if u.UserID == userID {
				found = u
			}
		}
		require.NotNil(t, found)
		assert.NotNil(t, found.LastSeen,
			"a login with no subsequent write must still register as activity")
	})

	t.Run("EmptyUserIsANoOp", func(t *testing.T) {
		// Defensive: never fabricate an event with no initiator, mirroring how
		// recordEvent skips when there is no user in context.
		var before int
		require.NoError(t, conn.QueryRow(ctx,
			`SELECT count(*) FROM public.events WHERE resource='session'`).Scan(&before))
		require.NoError(t, RecordLogin(ctx, ""))
		var after int
		require.NoError(t, conn.QueryRow(ctx,
			`SELECT count(*) FROM public.events WHERE resource='session'`).Scan(&after))
		assert.Equal(t, before, after, "an empty user id must write nothing")
	})
}
