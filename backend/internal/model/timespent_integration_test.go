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

// TestFindTimeSpentIntegration pins the MEASURED (view-based) dwell math against
// the real SQL: a question view accrues time up to the next event by the same
// user in the same system+data call (a later view, or a save), every interval
// is clamped at the idle cap, a trailing view with no following event
// contributes nothing, and a view made in a read-only session accrues to viewer
// time rather than editor time. Unit tests over the generated SQL cannot see
// this - it depends on the LEAD window, the resource union of 'viewed' +
// 'public.scores', the LEAST clamp, and the readonly FILTER split all
// interacting over real rows.
//
// Events carry no FK to datacalls/scores and so are not swept by the
// datacall-prefix purge; this test deletes the events it inserts explicitly.
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestFindTimeSpentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// A synthetic data call gives a guaranteed-unused id (purged by prefix); a
	// real system anchors the scope filter; a real user satisfies events.userid's
	// FK to users.
	var dc int32
	suffix := time.Now().UnixNano()
	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), NOW() + INTERVAL '90 days')
		RETURNING datacallid
	`, fmt.Sprintf("%stimespent_%d", integrationTestPrefix, suffix)).Scan(&dc)
	require.NoError(t, err)

	var fismaSystemID int32
	err = conn.QueryRow(ctx, `SELECT fismasystemid FROM fismasystems LIMIT 1`).Scan(&fismaSystemID)
	require.NoError(t, err, "need at least one seeded system")

	var userID string
	err = conn.QueryRow(ctx, `SELECT userid FROM users LIMIT 1`).Scan(&userID)
	require.NoError(t, err, "need at least one seeded user for events.userid FK")

	// Events written directly with explicit timestamps so the gaps are exact.
	defer func() {
		_, _ = conn.Exec(context.Background(),
			`DELETE FROM events WHERE (payload->>'datacallid')::int = $1`, dc)
	}()

	base := time.Now().Add(-4 * time.Hour)
	insertView := func(questionID int32, readonly bool, at time.Time) {
		t.Helper()
		_, err := conn.Exec(ctx, `
			INSERT INTO events (userid, action, resource, createdat, payload)
			VALUES ($1, 'viewed', 'questionnaire', $2::timestamptz, $3::jsonb)
		`, userID, at, fmt.Sprintf(`{"fismasystemid":%d,"datacallid":%d,"questionid":%d,"readonly":%t}`, fismaSystemID, dc, questionID, readonly))
		require.NoError(t, err)
	}
	insertSave := func(at time.Time) {
		t.Helper()
		_, err := conn.Exec(ctx, `
			INSERT INTO events (userid, action, resource, createdat, payload)
			VALUES ($1, 'updated', 'public.scores', $2::timestamptz, $3::jsonb)
		`, userID, at, fmt.Sprintf(`{"fismasystemid":%d,"datacallid":%d,"scoreid":999999}`, fismaSystemID, dc))
		require.NoError(t, err)
	}

	// Timeline for one user on one system:
	//   q1 viewed (editor) -> next is q2 60s later          => 60s editor
	//   q2 viewed (viewer) -> next is q3 2h later, CLAMPED   => idleCapSeconds viewer
	//   q3 viewed (editor) -> next is a save 30s later       => 30s editor
	//   q4 viewed (editor) -> no following event             => excluded
	insertView(1, false, base)
	insertView(2, true, base.Add(60*time.Second))
	insertView(3, false, base.Add(60*time.Second+2*time.Hour))
	insertSave(base.Add(60*time.Second + 2*time.Hour + 30*time.Second))
	insertView(4, false, base.Add(60*time.Second+2*time.Hour+60*time.Second))

	rows, err := FindTimeSpent(ctx, FindTimeSpentInput{
		DataCallID:    &dc,
		FismaSystemID: &fismaSystemID,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1, "single-system filter must return exactly that system")

	ts := rows[0]
	assert.Equal(t, fismaSystemID, ts.FismaSystemID)

	// q1=60 editor, q2=clamp viewer, q3=30 editor, q4=excluded (no trailing event).
	wantEditor := float64(60 + 30)
	wantViewer := float64(idleCapSeconds)
	wantTotal := wantEditor + wantViewer
	assert.Equal(t, wantTotal, ts.TotalSeconds, "60 editor + clamped q2 viewer + 30 editor, trailing view excluded")
	assert.Equal(t, wantEditor, ts.EditorSeconds, "q1 + q3 are editor views")
	assert.Equal(t, wantViewer, ts.ViewerSeconds, "q2 is a read-only view -> viewer time")
	assert.Equal(t, int32(3), ts.QuestionsMeasured, "q1, q2, q3 measured; q4 has no following event")
	assert.InDelta(t, wantTotal/3, ts.AverageSecondsPerQuestion, 0.001, "average is total over distinct measured questions")

	require.Len(t, ts.PerPerson, 1)
	assert.Equal(t, userID, ts.PerPerson[0].UserID)
	assert.Equal(t, wantTotal, ts.PerPerson[0].TotalSeconds)
	assert.Equal(t, wantEditor, ts.PerPerson[0].EditorSeconds)
	assert.Equal(t, wantViewer, ts.PerPerson[0].ViewerSeconds)
	assert.Equal(t, int32(3), ts.PerPerson[0].QuestionsMeasured)

	// Per-question breakdown: one person per question, avg == that person's time.
	require.Len(t, ts.PerQuestion, 3)
	for _, q := range ts.PerQuestion {
		assert.Equal(t, int32(1), q.People, "one user touched each question")
	}
}
