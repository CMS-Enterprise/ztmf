package model

import (
	"context"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FindEvents pages the audit trail for the admin events page (ztmf#564). The
// property that matters is a total order: bulk writers stamp batches of rows
// with identical createdat values, and without the eventid tiebreaker tied
// rows may swap between requests, duplicating or skipping rows across page
// boundaries. The fixture therefore deliberately writes tied timestamps and
// asserts that walking the pages covers every row exactly once.
func TestFindEventsPaginationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)

	// A resource name no writer uses keeps the fixture invisible to every
	// other reader and makes cleanup a single predicate. Registered before
	// any insert so every exit path cleans up; see events_login test for why
	// this is a Cleanup, not a defer.
	const resource = "pagination_test"
	t.Cleanup(func() {
		_, _ = conn.Exec(ctx, `DELETE FROM public.events WHERE resource=$1`, resource)
		conn.Release()
	})

	// events.userid is a FK, so borrow any seeded user as the initiator.
	var userID string
	require.NoError(t, conn.QueryRow(ctx, `SELECT userid FROM public.users LIMIT 1`).Scan(&userID))

	// Five rows sharing one timestamp (the case createdat alone cannot order)
	// plus one strictly older row, all in the far past so they cannot collide
	// with rows other tests write while this one runs.
	tied := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	older := tied.Add(-time.Hour)
	for _, ts := range []time.Time{tied, tied, tied, tied, tied, older} {
		_, err := conn.Exec(ctx,
			`INSERT INTO public.events (userid, action, resource, createdat, payload) VALUES ($1, 'created', $2, $3, '{}')`,
			userID, resource, ts)
		require.NoError(t, err)
	}

	res := resource
	limit := uint32(2)

	// Walk the pages and require exactly-once coverage in a strictly
	// decreasing (createdat, eventid) order.
	seen := map[int64]bool{}
	var prev *Event
	for offset := uint32(0); ; offset += limit {
		off := offset
		page, err := FindEvents(ctx, &FindEventsInput{Resource: &res, Limit: &limit, Offset: &off})
		require.NoError(t, err)
		assert.EqualValues(t, 6, page.Total, "total counts all matches, not the page")
		assert.Equal(t, limit, page.Limit, "applied limit is echoed")
		assert.Equal(t, off, page.Offset, "applied offset is echoed")
		if len(page.Events) == 0 {
			break
		}
		for _, e := range page.Events {
			require.Positive(t, e.EventID, "backfilled identity must be present on read")
			require.False(t, seen[e.EventID], "row %d appeared on two pages", e.EventID)
			seen[e.EventID] = true
			if prev != nil {
				after := e.CreatedAt.After(*prev.CreatedAt)
				tie := e.CreatedAt.Equal(*prev.CreatedAt) && e.EventID > prev.EventID
				require.False(t, after || tie, "order must be createdat DESC, eventid DESC")
			}
			prev = e
		}
	}
	assert.Len(t, seen, 6, "paging must cover every row exactly once")

	// Inclusive time bounds: from=tied keeps the tied five and drops the
	// older row; to=older keeps only the older row.
	page, err := FindEvents(ctx, &FindEventsInput{Resource: &res, From: &tied})
	require.NoError(t, err)
	assert.EqualValues(t, 5, page.Total)
	assert.Len(t, page.Events, 5, "five rows fit in the default limit")

	page, err = FindEvents(ctx, &FindEventsInput{Resource: &res, To: &older})
	require.NoError(t, err)
	assert.EqualValues(t, 1, page.Total)

	// A range matching nothing is an empty page, not an error, and the
	// events list must be non-nil so it serializes as [] rather than null.
	beforeAll := older.Add(-time.Hour)
	page, err = FindEvents(ctx, &FindEventsInput{Resource: &res, To: &beforeAll})
	require.NoError(t, err)
	assert.EqualValues(t, 0, page.Total)
	assert.NotNil(t, page.Events)
	assert.Empty(t, page.Events)
}
