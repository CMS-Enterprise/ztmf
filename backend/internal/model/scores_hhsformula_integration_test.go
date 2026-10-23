package model

import (
	"context"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPillarScoreHHSFormulaIntegration recomputes every pillar score the
// aggregate returns from the raw answers and checks it against the HHS
// scorecard formula (ztmf-misc#433): the average of the 1-4 answers times
// 1.25, an unanswered question counting as 1, and a pillar with no answers at
// all pinned to 1.0 (Not Assessed). The system score must stay the plain
// average of its pillar scores.
//
// Requires DB_* env vars pointing at a seeded ZTMF database. Skipped under
// `go test -short`.
func TestPillarScoreHHSFormulaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	aggs, err := FindScoresAggregate(ctx, FindScoresInput{IncludePillars: boolPtr(true)})
	require.NoError(t, err)
	require.NotEmpty(t, aggs, "expected at least one aggregate from seeded data")

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	for _, a := range aggs {
		sum := 0.0
		for _, p := range a.PillarScores {
			var expected, answered int
			var total float64
			err := conn.QueryRow(ctx, `
				SELECT COUNT(*), COUNT(fo.score), COALESCE(SUM(COALESCE(fo.score, 1)), 0)
				FROM fismasystems fs
				JOIN datacenterenvironments d ON d.datacenterenvironment = fs.datacenterenvironment
				JOIN functions f ON f.datacenterenvironment = d.scoring_key
				JOIN questions q ON q.questionid = f.questionid
				LEFT JOIN scores s
				  ON s.fismasystemid = fs.fismasystemid
				 AND s.datacallid    = $2
				 AND s.functionid    = f.functionid
				LEFT JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
				WHERE fs.fismasystemid = $1 AND q.pillarid = $3`,
				a.FismaSystemID, a.DataCallID, p.PillarID,
			).Scan(&expected, &answered, &total)
			require.NoError(t, err)
			require.Greater(t, expected, 0, "pillar %s has questions", p.Pillar)

			want := 1.0
			if answered > 0 {
				want = total / float64(expected) * 1.25
			}
			assert.InDelta(t, want, p.Score, 1e-9,
				"pillar %s for system %d / datacall %d: %d of %d answered",
				p.Pillar, a.FismaSystemID, a.DataCallID, answered, expected)
			sum += p.Score
		}
		if len(a.PillarScores) > 0 {
			assert.InDelta(t, sum/float64(len(a.PillarScores)), a.SystemScore, 1e-9,
				"system score is the average of its pillars for system %d / datacall %d",
				a.FismaSystemID, a.DataCallID)
		}
	}
}
