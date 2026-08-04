package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LastSeen answers "has this account ever been used" for the users list. It is
// derived on read from the events audit log rather than stored, so these tests
// pin the derivation against real event rows: a user with activity reports the
// newest one, a user with none reports nil, and the single-user path (which
// runs in the auth middleware on every request) deliberately does not pay for
// the lookup at all.
func TestFindUsersLastSeenIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	users, err := FindUsers(ctx, &FindUsersInput{})
	require.NoError(t, err)
	require.NotEmpty(t, users, "seeded database should have users")

	byID := map[string]*User{}
	for _, u := range users {
		byID[u.UserID] = u
	}

	t.Run("MatchesNewestEventForUsersWithActivity", func(t *testing.T) {
		// Resolve the subject from the events table rather than hardcoding an
		// id, so the test cannot pass vacuously against a fixture with no
		// events at all.
		var userID string
		err := conn.QueryRow(ctx, `
			SELECT userid FROM public.events
			GROUP BY userid ORDER BY count(*) DESC LIMIT 1
		`).Scan(&userID)
		require.NoError(t, err, "fixture must contain at least one event")

		u, ok := byID[userID]
		require.True(t, ok, "the busiest event author should appear in the users list")
		require.NotNil(t, u.LastSeen, "a user with events must report a last_seen")

		var expected any
		require.NoError(t, conn.QueryRow(ctx,
			`SELECT MAX(createdat) FROM public.events WHERE userid = $1`, userID).Scan(&expected))
		assert.Equal(t, expected, *u.LastSeen,
			"last_seen must be the newest event for that user, not any other row")
	})

	t.Run("NilForUsersWithNoActivity", func(t *testing.T) {
		var userID string
		err := conn.QueryRow(ctx, `
			SELECT u.userid FROM public.users u
			WHERE NOT COALESCE(u.deleted,false)
			  AND NOT EXISTS (SELECT 1 FROM public.events e WHERE e.userid = u.userid)
			LIMIT 1
		`).Scan(&userID)
		if err != nil {
			t.Skip("fixture has no user without events; nothing to assert here")
		}

		u, ok := byID[userID]
		require.True(t, ok)
		assert.Nil(t, u.LastSeen,
			"a user who has recorded no action must report null, not a zero time")
	})

	t.Run("SingleUserPathSkipsTheLookup", func(t *testing.T) {
		// findUser runs in the auth middleware on every authenticated request,
		// so it must not pay for a correlated subquery that only the list view
		// renders. Lax scanning leaves the field nil; this pins that choice so
		// a later "make it consistent" edit is a deliberate one.
		var userID string
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT userid FROM public.events GROUP BY userid ORDER BY count(*) DESC LIMIT 1
		`).Scan(&userID))

		one, err := FindUserByID(ctx, userID)
		require.NoError(t, err)
		require.NotNil(t, one)
		assert.Nil(t, one.LastSeen,
			"findUser must not populate last_seen; it is a list-only field")
	})
}
