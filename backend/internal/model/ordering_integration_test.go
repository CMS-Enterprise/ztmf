package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The questionnaire's order used to live only in the frontend (ztmf-ui's
// PILLAR_ORDER / PILLAR_FUNCTION_MAP), because pillars.ordr, questions.ordr and
// functions.ordr were 0 on every row. Migration 0056 and the functions.ordr
// backfill that followed it move that order into the data, and the read paths
// sort by it with a questionid tiebreaker.
// Since ztmf-misc#393 deleted the client-side sort, this response IS the order
// the questionnaire renders in.
//
// These tests pin the property that survives regardless of which fixture the
// database was seeded from: the same call twice returns the same sequence, and
// a pillar's questions arrive as one contiguous block rather than interleaved.
// That is the guarantee an unordered query cannot make - it returns heap order,
// which is stable only until a row is rewritten (the ztmf-misc#279 failure).
//
// The curated CISA ranks are still not asserted here. The empire seed loads
// after migrations run, so the ordr backfills match none of its fictional
// function names; it carries its own ranks instead (see _test_data_empire.sql),
// which are deliberately its pillarid sequence rather than the CISA one. Those
// migrations' own SQL is covered directly in the migrations package, on
// canonically-named fixture rows.

func TestFindQuestionsByFismaSystemOrderingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	// A system whose datacenterenvironment resolves to a multi-pillar
	// questionnaire, so "grouped by pillar" is a meaningful assertion.
	var fismaSystemID int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT fs.fismasystemid
		FROM fismasystems fs
		JOIN datacenterenvironments dce ON dce.datacenterenvironment = fs.datacenterenvironment
		JOIN functions f ON f.datacenterenvironment = dce.scoring_key
		JOIN questions q ON q.questionid = f.questionid
		GROUP BY fs.fismasystemid
		HAVING count(DISTINCT q.pillarid) > 1
		ORDER BY fs.fismasystemid
		LIMIT 1
	`).Scan(&fismaSystemID), "seed must contain a system with a multi-pillar questionnaire")

	first, err := FindQuestionsByFismaSystem(ctx, fismaSystemID, FindQuestionsByFismaSystemInput{})
	require.NoError(t, err)
	require.NotEmpty(t, first)

	second, err := FindQuestionsByFismaSystem(ctx, fismaSystemID, FindQuestionsByFismaSystemInput{})
	require.NoError(t, err)

	require.Equal(t, questionOrderKeys(first), questionOrderKeys(second),
		"two consecutive calls must return the same question sequence")

	assertPillarsContiguous(t, len(first), func(i int) any { return first[i].Pillar.PillarID })

	// Every pillar the questionnaire returns must carry a real rank. Without
	// this the sort below passes vacuously on an all-zero catalog, which is
	// exactly the state ztmf-misc#393 exists to end.
	for i, q := range first {
		assert.NotZerof(t, q.Pillar.Order,
			"row %d: pillar %q is unranked; the catalog's ordering is not in the database",
			i, q.Pillar.Pillar)
	}

	// The declared sort key must be non-decreasing across the result: whatever
	// order the rows are in, it is the order the ORDER BY asked for.
	for i := 1; i < len(first); i++ {
		prev := []int{first[i-1].Pillar.Order, derefInt(first[i-1].Ordr), derefInt(first[i-1].Function.Ordr), int(first[i-1].QuestionID)}
		cur := []int{first[i].Pillar.Order, derefInt(first[i].Ordr), derefInt(first[i].Function.Ordr), int(first[i].QuestionID)}
		assert.LessOrEqualf(t, sortKeyCompare(prev, cur), 0,
			"row %d breaks the (pillars.ordr, questions.ordr, functions.ordr, questionid) sort: %v then %v",
			i, prev, cur)
	}
}

func TestFindAnswersOrderingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err, "DB connection required for integration test; ensure DB_* env vars are set")
	defer conn.Release()

	var dataCallID int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT datacallid FROM scores GROUP BY datacallid ORDER BY count(*) DESC LIMIT 1
	`).Scan(&dataCallID), "seed must contain at least one scored data call")

	first, err := FindAnswers(ctx, FindAnswersInput{DataCallID: dataCallID})
	require.NoError(t, err)
	require.NotEmpty(t, first)

	second, err := FindAnswers(ctx, FindAnswersInput{DataCallID: dataCallID})
	require.NoError(t, err)

	require.Equal(t, answerOrderKeys(first), answerOrderKeys(second),
		"two consecutive exports must emit rows in the same order")

	// Systems are the outermost grouping, pillars the next one inside each
	// system. Both must be contiguous or the spreadsheet interleaves.
	assertPillarsContiguous(t, len(first), func(i int) any { return first[i].FismaSystemID })
	assertPillarsContiguous(t, len(first), func(i int) any {
		return [2]any{first[i].FismaSystemID, first[i].Pillar}
	})
}

// assertPillarsContiguous fails if a key value reappears after a different key
// value has intervened, i.e. the grouping is interleaved rather than blocked.
func assertPillarsContiguous(t *testing.T, n int, keyAt func(int) any) {
	t.Helper()

	seen := map[any]bool{}
	var current any
	for i := 0; i < n; i++ {
		k := keyAt(i)
		if i == 0 || k != current {
			assert.Falsef(t, seen[k], "group %v is split across the result at row %d", k, i)
			seen[k] = true
			current = k
		}
	}
}

// sortKeyCompare returns -1, 0 or 1 for the lexicographic order of two sort
// keys. Written out because testify's ordering assertions only handle scalars.
func sortKeyCompare(a, b []int) int {
	for i := range a {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}

func questionOrderKeys(qs []*Question) []int32 {
	keys := make([]int32, len(qs))
	for i, q := range qs {
		keys[i] = q.QuestionID
	}
	return keys
}

// Notes is part of the key on purpose: a duplicate score row on one function
// (ztmf#491) ties on every other column, so a key of acronym/pillar/function
// alone compares equal however the two copies happen to come back and cannot
// detect the instability scoreid in the ORDER BY exists to prevent.
func answerOrderKeys(as []*Answer) [][4]string {
	keys := make([][4]string, len(as))
	for i, a := range as {
		var notes string
		if a.Notes != nil {
			notes = *a.Notes
		}
		keys[i] = [4]string{a.FismaAcronym, a.Pillar, a.Function, notes}
	}
	return keys
}
