package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Emberfall ISSO from the empire seed, granted Shield Gen (1003) only.
const emberfallISSOUserID = "66666666-6666-6666-6666-666666666666"

func systemsOf(rows []*SystemInsight) map[int32]bool {
	m := make(map[int32]bool, len(rows))
	for _, r := range rows {
		m[r.FismaSystemID] = true
	}
	return m
}

// TestFindSystemInsightsIntegration pins the /insights read behavior against the
// real SQL and empire seed (issue #416): the users_fismasystems scope, the
// fail-closed OpDiv restriction, and the insights_enabled OpDiv gate that hides
// rows for systems in a non-enabled OpDiv even from an unscoped admin. Unit tests
// over the generated SQL cannot see these - they depend on the joins against the
// seeded fismasystems / opdivs / users_fismasystems rows.
//
// Seed facts (backend/_test_data_empire.sql): system_insights rows at questionid
// 1 exist for 1001 (Death Star, EMPIRE/enabled), 1003 (Shield Gen, EMPIRE/enabled,
// assigned to emberfallISSOUserID) and 1005 (RB-1, REBELLION/NOT enabled). The
// Emberfall ISSO is granted 1003 only.
//
// Requires DB_* env vars pointing at the seeded ephemeral test DB (make
// test-integration). Skipped under `go test -short`.
func TestFindSystemInsightsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	q1 := int32(1)
	isso := emberfallISSOUserID

	// Unscoped admin: sees both enabled systems at q1; the OpDiv gate hides the
	// REBELLION system (1005, insights_enabled = FALSE) entirely.
	admin, err := FindSystemInsights(ctx, FindSystemInsightsInput{QuestionID: &q1})
	require.NoError(t, err)
	got := systemsOf(admin)
	assert.True(t, got[1001], "admin should see enabled system 1001 at q1")
	assert.True(t, got[1003], "admin should see enabled system 1003 at q1")
	assert.False(t, got[1005], "OpDiv gate must hide 1005 (insights_enabled = FALSE) even for an unscoped admin")

	// Payload is served as opaque JSON: confirm a seeded key round-trips (proves
	// the jsonb column is passed through, which Emberfall cannot assert on a list).
	for _, r := range admin {
		if r.FismaSystemID == 1003 {
			assert.Contains(t, string(r.Payload), "shield-relay-without-mfa", "payload jsonb should pass through untouched")
		}
	}

	// ISSO scoped to grants: assigned to 1003, so sees 1003 but never 1001
	// (enabled but unassigned) or 1005 (gated).
	issoRows, err := FindSystemInsights(ctx, FindSystemInsightsInput{QuestionID: &q1, UserID: &isso})
	require.NoError(t, err)
	scoped := systemsOf(issoRows)
	assert.True(t, scoped[1003], "ISSO assigned to 1003 should see its insight row")
	assert.False(t, scoped[1001], "ISSO must NOT see an enabled but unassigned system (1001)")
	assert.False(t, scoped[1005], "ISSO must NOT see a gated system (1005)")

	// ISSO filtered to an unassigned system -> empty (scoped out, no leak).
	unassigned := int32(1001)
	rows, err := FindSystemInsights(ctx, FindSystemInsightsInput{FismaSystemID: &unassigned, UserID: &isso})
	require.NoError(t, err)
	assert.Empty(t, rows, "ISSO must get no rows for an unassigned system")

	// Fail-closed OpDiv tier: restrict with no granted OpDivs -> empty.
	failClosed, err := FindSystemInsights(ctx, FindSystemInsightsInput{QuestionID: &q1, RestrictToOpDivIDs: true})
	require.NoError(t, err)
	assert.Empty(t, failClosed, "RestrictToOpDivIDs with empty OpDivIDs must fail closed to no rows")
}
