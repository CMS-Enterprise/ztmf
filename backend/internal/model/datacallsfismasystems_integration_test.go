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

// TestFindFismaSystemDataCallsOrderIntegration pins the #393 sibling cleanup
// flagged in the #397 review: a system's completed-datacalls list must order by
// deadline, not datecreated. A backfilled historical call is INSERTed at import
// time, so its datecreated is recent even though its deadline is long past -
// under datecreated ordering it would sort above the real current cycle.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindFismaSystemDataCallsOrderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	suffix := time.Now().UnixNano()
	const systemID = int32(1001) // Death Star: stable empire-seed system with existing junction rows

	// The "current" cycle: created a while ago, deadline far out (beats the
	// empire seed's furthest-out 2099 cycle so it must sort first overall).
	var currentID int32
	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW() - INTERVAL '30 days', '2102-01-01T00:00:00Z'::timestamptz)
		RETURNING datacallid
	`, fmt.Sprintf("%scurrent_%d", integrationTestPrefix, suffix)).Scan(&currentID)
	require.NoError(t, err)

	// The backfilled historical cycle: created just now (as an import would
	// be), deadline long past. Under the old datecreated ordering this row
	// wrongly sorts to the top of the system's list.
	var backfilledID int32
	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), '2019-12-31T23:59:59Z'::timestamptz)
		RETURNING datacallid
	`, fmt.Sprintf("%sbackfilled_%d", integrationTestPrefix, suffix)).Scan(&backfilledID)
	require.NoError(t, err)

	_, err = conn.Exec(ctx, `
		INSERT INTO datacalls_fismasystems (datacallid, fismasystemid)
		VALUES ($1, $3), ($2, $3) ON CONFLICT DO NOTHING
	`, currentID, backfilledID, systemID)
	require.NoError(t, err)

	dataCalls, err := FindFismaSystemDataCalls(ctx, systemID)
	require.NoError(t, err)
	require.NotEmpty(t, dataCalls)

	// The far-future current cycle leads the list; the freshly-created
	// backfilled cycle must NOT ride its datecreated to the top.
	assert.Equal(t, currentID, dataCalls[0].DataCallID,
		"the furthest-out deadline must sort first, regardless of datecreated")

	// And the whole list is non-increasing by deadline, so the backfilled
	// row sits in its chronological place at the bottom, not merely "not first".
	for i := 1; i < len(dataCalls); i++ {
		assert.False(t, dataCalls[i].Deadline.After(dataCalls[i-1].Deadline),
			"deadlines must be non-increasing: position %d (%s) sorts after position %d (%s)",
			i, dataCalls[i].DataCall, i-1, dataCalls[i-1].DataCall)
	}
	assert.Equal(t, backfilledID, dataCalls[len(dataCalls)-1].DataCallID,
		"the 2019-deadline backfill must sort last despite the newest datecreated")
}
