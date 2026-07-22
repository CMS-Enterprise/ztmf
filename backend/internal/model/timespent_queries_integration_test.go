package model

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// timeSpentQueriesPath is the delivered SQL doc, relative to this package dir.
const timeSpentQueriesPath = "../../docs/timespent_queries.sql"

// extractQuery pulls the single SQL statement that begins at the given marker
// comment (e.g. "-- M1.") and runs to its terminating semicolon, so these tests
// execute the EXACT queries shipped in docs/timespent_queries.sql. It strips
// "--" line comments before scanning, so a ';' inside a comment is not mistaken
// for the end of the statement.
func extractQuery(t *testing.T, content, marker string) string {
	t.Helper()
	start := strings.Index(content, marker)
	require.GreaterOrEqual(t, start, 0, "marker %q not found in %s", marker, timeSpentQueriesPath)

	var b strings.Builder
	for _, line := range strings.Split(content[start:], "\n") {
		code := line
		if i := strings.Index(code, "--"); i >= 0 {
			code = code[:i] // drop the comment portion
		}
		b.WriteString(code)
		b.WriteString("\n")
		if strings.Contains(code, ";") {
			break
		}
	}
	stmt := b.String()
	end := strings.Index(stmt, ";")
	require.GreaterOrEqual(t, end, 0, "no terminating ';' after marker %q", marker)
	return stmt[:end+1]
}

// TestTimeSpentQueriesMeasuredIntegration runs the shipped MEASURED "per system"
// query (M1) against seeded 'viewed' events and pins the dwell math:
// view->next-view intervals, the 30-minute clamp, the editor/viewer split by
// readonly, the questions-existence join, and - critically - that a score SAVE
// interleaved between two views does NOT truncate a view's dwell (the
// view-only-boundary fix).
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`. Events carry no FK cascade, so inserts are deleted here.
func TestTimeSpentQueriesMeasuredIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required; ensure DB_* env vars are set")
	defer conn.Release()

	sqlDoc, err := os.ReadFile(timeSpentQueriesPath)
	require.NoError(t, err, "read %s", timeSpentQueriesPath)

	var dc int32
	suffix := time.Now().UnixNano()
	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), NOW() + INTERVAL '90 days') RETURNING datacallid
	`, fmt.Sprintf("%stsq_measured_%d", integrationTestPrefix, suffix)).Scan(&dc)
	require.NoError(t, err)

	var fismaSystemID int32
	require.NoError(t, conn.QueryRow(ctx, `SELECT fismasystemid FROM fismasystems LIMIT 1`).Scan(&fismaSystemID))
	var userID string
	require.NoError(t, conn.QueryRow(ctx, `SELECT userid FROM users LIMIT 1`).Scan(&userID))

	qRows, err := conn.Query(ctx, `SELECT questionid FROM questions ORDER BY questionid LIMIT 3`)
	require.NoError(t, err)
	var qids []int32
	for qRows.Next() {
		var id int32
		require.NoError(t, qRows.Scan(&id))
		qids = append(qids, id)
	}
	qRows.Close()
	if len(qids) < 3 {
		t.Skip("seed has fewer than 3 questions; cannot exercise the measured scenario")
	}

	// A real score row so the interleaved save event resolves (and to clean up).
	var functionOptionID int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT fo.functionoptionid FROM functionoptions fo
		JOIN functions f ON f.functionid = fo.functionid LIMIT 1
	`).Scan(&functionOptionID))
	var scoreID int32
	require.NoError(t, conn.QueryRow(ctx, `INSERT INTO scores (fismasystemid, datacallid, functionoptionid) VALUES ($1,$2,$3) RETURNING scoreid`, fismaSystemID, dc, functionOptionID).Scan(&scoreID))

	defer func() {
		_, _ = conn.Exec(context.Background(), `DELETE FROM events WHERE (payload->>'datacallid')::int = $1`, dc)
		_, _ = conn.Exec(context.Background(), `DELETE FROM scores WHERE datacallid = $1`, dc)
	}()

	base := time.Now().Add(-4 * time.Hour)
	view := func(questionID int32, readonly bool, at time.Time) {
		_, err := conn.Exec(ctx, `
			INSERT INTO events (userid, action, resource, createdat, payload)
			VALUES ($1, 'viewed', 'questionnaire', $2::timestamptz, $3::jsonb)
		`, userID, at, fmt.Sprintf(`{"fismasystemid":%d,"datacallid":%d,"questionid":%d,"readonly":%t}`, fismaSystemID, dc, questionID, readonly))
		require.NoError(t, err)
	}
	save := func(at time.Time) {
		_, err := conn.Exec(ctx, `
			INSERT INTO events (userid, action, resource, createdat, payload)
			VALUES ($1, 'updated', 'public.scores', $2::timestamptz, $3::jsonb)
		`, userID, at, fmt.Sprintf(`{"fismasystemid":%d,"datacallid":%d,"scoreid":%d}`, fismaSystemID, dc, scoreID))
		require.NoError(t, err)
	}

	// q0 editor 60s -> q1 viewer clamped(2h)->1800s -> q2 editor 30s -> trailing.
	// A save is interleaved 30s into q0: under the view-only-boundary rule it must
	// NOT truncate q0's dwell (which stays 60s).
	view(qids[0], false, base)
	save(base.Add(30 * time.Second))
	view(qids[1], true, base.Add(60*time.Second))
	view(qids[2], false, base.Add(60*time.Second+2*time.Hour))
	view(qids[0], false, base.Add(60*time.Second+2*time.Hour+30*time.Second))

	q := strings.ReplaceAll(extractQuery(t, string(sqlDoc), "-- M1."), ":datacallid", fmt.Sprintf("%d", dc))

	var (
		gotSystem      int32
		editor, viewer *float64
		editorQ        int64
		avgEditor      *float64
	)
	err = conn.QueryRow(ctx, q).Scan(&gotSystem, &editor, &viewer, &editorQ, &avgEditor)
	require.NoError(t, err, "M1 query failed:\n%s", q)

	assert.Equal(t, fismaSystemID, gotSystem)
	require.NotNil(t, editor)
	require.NotNil(t, viewer)
	require.NotNil(t, avgEditor)
	assert.Equal(t, float64(90), *editor, "q0(60, unbroken by the interleaved save)+q2(30)")
	assert.Equal(t, float64(1800), *viewer, "q1 read-only view, clamped at 30m")
	assert.Equal(t, int64(2), editorQ, "q0 and q2 are editor questions (q1 is viewer)")
	assert.Equal(t, float64(45), *avgEditor, "editor 90s / 2 editor questions")
}

// TestTimeSpentQueriesProxyIntegration runs the shipped PROXY queries (P1 total,
// P3 per-question) against seeded 'public.scores' saves and pins the save-gap
// proxy: the gap PRECEDING a save is attributed to THAT save's own question
// (LAG), so the first save is dropped and each interval lands on the right
// question. Uses two DISTINCT questions so a mis-attribution would be visible.
func TestTimeSpentQueriesProxyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	purgeIntegrationTestRows(t)
	defer purgeIntegrationTestRows(t)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required; ensure DB_* env vars are set")
	defer conn.Release()

	sqlDoc, err := os.ReadFile(timeSpentQueriesPath)
	require.NoError(t, err, "read %s", timeSpentQueriesPath)

	var dc int32
	suffix := time.Now().UnixNano()
	err = conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline)
		VALUES ($1, NOW(), NOW() + INTERVAL '90 days') RETURNING datacallid
	`, fmt.Sprintf("%stsq_proxy_%d", integrationTestPrefix, suffix)).Scan(&dc)
	require.NoError(t, err)

	var fismaSystemID int32
	require.NoError(t, conn.QueryRow(ctx, `SELECT fismasystemid FROM fismasystems LIMIT 1`).Scan(&fismaSystemID))
	var userID string
	require.NoError(t, conn.QueryRow(ctx, `SELECT userid FROM users LIMIT 1`).Scan(&userID))

	// Two function options mapping to two DISTINCT questions.
	type foq struct {
		fo, q int32
	}
	var pairs []foq
	rows, err := conn.Query(ctx, `
		SELECT DISTINCT ON (f.questionid) fo.functionoptionid, f.questionid
		FROM functionoptions fo
		JOIN functions f ON f.functionid = fo.functionid
		ORDER BY f.questionid
		LIMIT 2
	`)
	require.NoError(t, err)
	for rows.Next() {
		var p foq
		require.NoError(t, rows.Scan(&p.fo, &p.q))
		pairs = append(pairs, p)
	}
	rows.Close()
	if len(pairs) < 2 {
		t.Skip("seed has fewer than 2 questions; cannot exercise distinct-question attribution")
	}
	qA, qB := pairs[0].q, pairs[1].q

	var scoreA, scoreB int32
	require.NoError(t, conn.QueryRow(ctx, `INSERT INTO scores (fismasystemid, datacallid, functionoptionid) VALUES ($1,$2,$3) RETURNING scoreid`, fismaSystemID, dc, pairs[0].fo).Scan(&scoreA))
	require.NoError(t, conn.QueryRow(ctx, `INSERT INTO scores (fismasystemid, datacallid, functionoptionid) VALUES ($1,$2,$3) RETURNING scoreid`, fismaSystemID, dc, pairs[1].fo).Scan(&scoreB))

	defer func() {
		_, _ = conn.Exec(context.Background(), `DELETE FROM events WHERE (payload->>'datacallid')::int = $1`, dc)
		_, _ = conn.Exec(context.Background(), `DELETE FROM scores WHERE datacallid = $1`, dc)
	}()

	base := time.Now().Add(-3 * time.Hour)
	save := func(scoreID int32, at time.Time) {
		_, err := conn.Exec(ctx, `
			INSERT INTO events (userid, action, resource, createdat, payload)
			VALUES ($1, 'updated', 'public.scores', $2::timestamptz, $3::jsonb)
		`, userID, at, fmt.Sprintf(`{"fismasystemid":%d,"datacallid":%d,"scoreid":%d}`, fismaSystemID, dc, scoreID))
		require.NoError(t, err)
	}

	// save(qA) -> save(qB) 45s later -> save(qA) 2h later. With LAG attribution:
	//   save(qA)#1 has no preceding save -> dropped.
	//   save(qB)  gets the 45s gap that preceded it  -> qB = 45s.
	//   save(qA)#2 gets the 2h gap (clamped) preceding it -> qA = 1800s.
	save(scoreA, base)
	save(scoreB, base.Add(45*time.Second))
	save(scoreA, base.Add(45*time.Second+2*time.Hour))

	// P1: totals.
	p1 := strings.ReplaceAll(extractQuery(t, string(sqlDoc), "-- P1."), ":datacallid", fmt.Sprintf("%d", dc))
	var (
		gotSystem int32
		editor    *float64
		editorQ   int64
		avgEditor *float64
	)
	require.NoError(t, conn.QueryRow(ctx, p1).Scan(&gotSystem, &editor, &editorQ, &avgEditor), "P1 query failed:\n%s", p1)
	assert.Equal(t, fismaSystemID, gotSystem)
	require.NotNil(t, editor)
	assert.Equal(t, float64(1845), *editor, "45s (to qB) + clamped 1800s (to qA); first save dropped")
	assert.Equal(t, int64(2), editorQ, "two distinct questions measured")

	// P3: attribution - the 45s lands on qB, the clamped 1800s lands on qA.
	p3 := strings.ReplaceAll(extractQuery(t, string(sqlDoc), "-- P3."), ":datacallid", fmt.Sprintf("%d", dc))
	prows, err := conn.Query(ctx, p3)
	require.NoError(t, err, "P3 query failed:\n%s", p3)
	got := map[int32]float64{}
	for prows.Next() {
		var sys, qid int32
		var question string
		var people int64
		var avg *float64
		require.NoError(t, prows.Scan(&sys, &qid, &question, &people, &avg))
		require.NotNil(t, avg)
		got[qid] = *avg
	}
	prows.Close()
	assert.Equal(t, float64(45), got[qB], "the 45s gap before save(qB) belongs to qB")
	assert.Equal(t, float64(1800), got[qA], "the clamped 2h gap before save(qA)#2 belongs to qA")
}
