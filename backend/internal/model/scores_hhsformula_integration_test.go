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
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	// The seed gives every pillar a single question, so no pillar can be
	// partially answered. Make one: add an unanswered second question to a
	// pillar that a system has answered. Removed again on cleanup.
	var sysID, dcID, pillarID, answer int32
	var scoringKey string
	err = conn.QueryRow(ctx, `
		SELECT s.fismasystemid, s.datacallid, q.pillarid, fo.score, d.scoring_key
		FROM scores s
		JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
		JOIN functions f ON f.functionid = fo.functionid
		JOIN questions q ON q.questionid = f.questionid
		JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		JOIN datacenterenvironments d ON d.datacenterenvironment = fs.datacenterenvironment
		WHERE d.scoring_key = f.datacenterenvironment
		  AND NOT EXISTS (SELECT 1 FROM reducedpillarscopes r
		                  WHERE r.scoring_key = d.scoring_key AND r.pillarid = q.pillarid
		                    AND r.effective_datacallid <= s.datacallid)
		  AND (SELECT COUNT(*) FROM functions f2 JOIN questions q2 ON q2.questionid = f2.questionid
		       WHERE f2.datacenterenvironment = d.scoring_key AND q2.pillarid = q.pillarid) = 1
		ORDER BY s.scoreid LIMIT 1`,
	).Scan(&sysID, &dcID, &pillarID, &answer, &scoringKey)
	require.NoError(t, err, "seed must hold an answered single-question pillar to build the fixture from")

	var questionID, functionID int32
	require.NoError(t, conn.QueryRow(ctx,
		`INSERT INTO questions (question, notesprompt, pillarid) VALUES ('ztmf-misc#433 fixture', '', $1)
		 RETURNING questionid`, pillarID,
	).Scan(&questionID))
	t.Cleanup(func() {
		deleteFixtureRow(t, `DELETE FROM questions WHERE questionid = $1`, questionID)
	})
	require.NoError(t, conn.QueryRow(ctx,
		`INSERT INTO functions (function, datacenterenvironment, questionid, pillarid)
		 VALUES ('ztmf-misc#433 fixture', $1, $2, $3) RETURNING functionid`,
		scoringKey, questionID, pillarID,
	).Scan(&functionID))
	// Registered after the question's cleanup, so it runs first.
	t.Cleanup(func() {
		deleteFixtureRow(t, `DELETE FROM functions WHERE functionid = $1`, functionID)
	})

	aggs, err := FindScoresAggregate(ctx, FindScoresInput{IncludePillars: boolPtr(true)})
	require.NoError(t, err)
	require.NotEmpty(t, aggs, "expected at least one aggregate from seeded data")

	// The existing answer plus the new unanswered question counting as 1.
	fixtureWant := (float64(answer) + 1) / 2 * 1.25
	fixtureSeen := false
	for _, a := range aggs {
		if a.FismaSystemID != sysID || a.DataCallID != dcID {
			continue
		}
		for _, p := range a.PillarScores {
			if p.PillarID == pillarID {
				fixtureSeen = true
				assert.InDelta(t, fixtureWant, p.Score, 1e-9, "one answer of %d and one unanswered question", answer)
			}
		}
	}
	require.True(t, fixtureSeen, "fixture pillar must appear in the aggregate")

	// Both edge branches must be exercised, or the test proves only the
	// fully answered case.
	partial, empty := 0, 0
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

			switch {
			case answered == 0:
				empty++
			case answered < expected:
				partial++
			}
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
	assert.Positive(t, partial, "seed must include a partially answered pillar")
	assert.Positive(t, empty, "seed must include a pillar with nothing answered")
}

// deleteFixtureRow removes a fixture row on cleanup. It takes its own
// connection: cleanups run after the test body's deferred Release.
func deleteFixtureRow(t *testing.T, sql string, id int32) {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Errorf("cleanup: connect: %v", err)
		return
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, sql, id); err != nil {
		t.Errorf("cleanup: %v", err)
	}
}
