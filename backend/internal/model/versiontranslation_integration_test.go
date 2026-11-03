package model

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pinning a data call to a version, and carrying answers across versions
// (ztmf-misc#397). The fixtures copy seeded call 103's answers, all on the
// baseline, into open integration_test_ calls.

const repinSourceCall = 103

// openCallWithAnswers creates an open call pinned to version 1 holding call
// 103's answers as carried forward: not_started, no history.
func openCallWithAnswers(t *testing.T, ctx context.Context, conn *pgxpool.Conn, label string, deadline string) int32 {
	t.Helper()
	var dc int32
	require.NoError(t, conn.QueryRow(ctx, `
		INSERT INTO datacalls (datacall, datecreated, deadline, questionnaire_versionid)
		VALUES ($1, NOW(), $2::timestamptz, 1) RETURNING datacallid`,
		fmt.Sprintf("%s%s_%d", integrationTestPrefix, label, time.Now().UnixNano()), deadline).Scan(&dc))
	tag, err := conn.Exec(ctx, `
		INSERT INTO scores (fismasystemid, datecalculated, notes, notes_is_ai_summary, functionoptionid, datacallid, status)
		SELECT fismasystemid, datecalculated, notes, notes_is_ai_summary, functionoptionid, $2, 'not_started'
		  FROM scores WHERE datacallid = $1`, repinSourceCall, dc)
	require.NoError(t, err)
	require.Positive(t, tag.RowsAffected())
	return dc
}

// mostAnswered is the (lineage, score) most of a call's answers sit on, so a
// fork without it has something to drop.
func mostAnswered(t *testing.T, ctx context.Context, conn *pgxpool.Conn, dc int32) (root, score, answers int32) {
	t.Helper()
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT f.root_functionid, fo.score, COUNT(*)
		  FROM scores s
		  JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
		  JOIN functions f ON f.functionid = fo.functionid
		 WHERE s.datacallid = $1
		 GROUP BY 1, 2 ORDER BY 3 DESC, 1, 2 LIMIT 1`, dc).Scan(&root, &score, &answers))
	return root, score, answers
}

// withoutOption drops a (lineage, score) option from a published fork, as
// though the version had been published without it.
func withoutOption(t *testing.T, ctx context.Context, conn *pgxpool.Conn, v, root, score int32) {
	t.Helper()
	tag, err := conn.Exec(ctx, `
		DELETE FROM functionoptions fo USING functions f
		 WHERE f.functionid = fo.functionid AND f.versionid = $1 AND f.root_functionid = $2 AND fo.score = $3`, v, root, score)
	require.NoError(t, err)
	require.Positive(t, tag.RowsAffected())
}

// callState is everything a translation must leave alone, keyed by lineage
// so it compares across versions, plus where each answer now points.
type callState struct {
	rows     []string
	versions []int32
}

func readCallState(t *testing.T, ctx context.Context, conn *pgxpool.Conn, dc int32) callState {
	t.Helper()
	rows, err := conn.Query(ctx, `
		SELECT format('%s|%s|%s|%s|%s|%s|%s', s.fismasystemid, f.root_functionid, fo.score,
		              s.notes, s.notes_is_ai_summary, s.status, s.last_updated_at),
		       fo.versionid
		  FROM scores s
		  JOIN functionoptions fo ON fo.functionoptionid = s.functionoptionid
		  JOIN functions f ON f.functionid = s.functionid
		 WHERE s.datacallid = $1`, dc)
	require.NoError(t, err)
	defer rows.Close()
	var st callState
	seen := map[int32]bool{}
	for rows.Next() {
		var row string
		var v int32
		require.NoError(t, rows.Scan(&row, &v))
		st.rows = append(st.rows, row)
		if !seen[v] {
			seen[v] = true
			st.versions = append(st.versions, v)
		}
	}
	require.NoError(t, rows.Err())
	sort.Strings(st.rows)
	return st
}

func TestDataCallCreatePinsAVersionIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)
	create := func(version int32) (*DataCall, error) {
		return (&DataCall{
			DataCall:               fmt.Sprintf("%screate_%d", integrationTestPrefix, time.Now().UnixNano()),
			Deadline:               time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
			QuestionnaireVersionID: version,
		}).Save(ctx)
	}

	dc, err := create(0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, dc.QuestionnaireVersionID, "no version named: the newest published, the baseline")

	v2 := forkIdenticalVersion(t, conn)
	dc, err = create(0)
	require.NoError(t, err)
	assert.Equal(t, v2, dc.QuestionnaireVersionID, "the newest published version once there is one")

	dc, err = create(1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, dc.QuestionnaireVersionID, "an explicit published version is accepted")

	t.Run("PutCannotMoveThePin", func(t *testing.T) {
		updated, err := (&DataCall{DataCallID: dc.DataCallID, DataCall: dc.DataCall, Deadline: dc.Deadline, QuestionnaireVersionID: v2}).Save(ctx)
		require.NoError(t, err)
		assert.EqualValues(t, 1, updated.QuestionnaireVersionID)
		stored, err := FindDataCallByID(ctx, dc.DataCallID)
		require.NoError(t, err)
		assert.EqualValues(t, 1, stored.QuestionnaireVersionID)
	})

	t.Run("RefusesADraftAnArchivedOrAMissingVersion", func(t *testing.T) {
		draft, _, _ := minimalDraft(t, ctx, conn, "AWS")
		_, err := create(draft)
		requireInvalidOn(t, err, "questionnaire_versionid", draft)

		_, err = ArchiveQuestionnaireVersion(ctx, 1)
		require.NoError(t, err)
		t.Cleanup(func() {
			deleteFixtureRow(t, `UPDATE questionnaire_versions SET state = 'published' WHERE versionid = $1`, 1)
		})
		_, err = create(1)
		requireInvalidOn(t, err, "questionnaire_versionid", 1)

		_, err = create(999999)
		requireInvalidOn(t, err, "questionnaire_versionid", 999999)
	})
}

func TestDataCallRepinIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)
	yes := true
	aggregate := func(dc int32) []*ScoreAggregate {
		rows, err := FindScoresAggregate(ctx, FindScoresInput{DataCallID: &dc, IncludePillars: &yes})
		require.NoError(t, err)
		return rows
	}
	progress := func(dc int32) []*ScoreProgress {
		rows, err := FindScoreProgress(ctx, FindScoreProgressInput{DataCallID: &dc})
		require.NoError(t, err)
		return rows
	}
	pin := func(dc int32) int32 {
		stored, err := FindDataCallByID(ctx, dc)
		require.NoError(t, err)
		return stored.QuestionnaireVersionID
	}

	identical := forkIdenticalVersion(t, conn)
	probe := openCallWithAnswers(t, ctx, conn, "repin_probe", "2100-01-01")
	root, score, lost := mostAnswered(t, ctx, conn, probe)
	missing := forkIdenticalVersion(t, conn)
	withoutOption(t, ctx, conn, missing, root, score)

	t.Run("NoAnswersIsAFreeSwap", func(t *testing.T) {
		var dc int32
		require.NoError(t, conn.QueryRow(ctx, `
			INSERT INTO datacalls (datacall, datecreated, deadline, questionnaire_versionid)
			VALUES ($1, NOW(), '2100-01-01', 1) RETURNING datacallid`, integrationTestPrefix+"repin_empty").Scan(&dc))
		res, err := RepinDataCall(ctx, dc, missing)
		require.NoError(t, err)
		assert.Equal(t, missing, res.QuestionnaireVersionID)
		assert.Zero(t, res.Translated)
		assert.Zero(t, res.Dropped)
	})

	t.Run("UntouchedAnswersRemapAndDrop", func(t *testing.T) {
		dc := openCallWithAnswers(t, ctx, conn, "repin_untouched", "2100-01-01")
		before := readCallState(t, ctx, conn, dc)
		res, err := RepinDataCall(ctx, dc, missing)
		require.NoError(t, err)
		assert.EqualValues(t, lost, res.Dropped, "answers on the removed option are dropped and counted")
		assert.EqualValues(t, len(before.rows)-int(lost), res.Translated)
		after := readCallState(t, ctx, conn, dc)
		assert.Equal(t, []int32{missing}, after.versions)
		assert.Len(t, after.rows, len(before.rows)-int(lost))
		assert.Subset(t, before.rows, after.rows, "what remains moved by lineage and score, nothing else changed")
		assert.Equal(t, missing, pin(dc))
	})

	t.Run("GivenAnswersThatTranslateMoveWithHistory", func(t *testing.T) {
		dc := openCallWithAnswers(t, ctx, conn, "repin_given", "2100-01-01")
		// One answer given this cycle, with history: an edit, so its head is an
		// update and undo would be offered before the re-pin.
		var edited Score
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT s.scoreid, s.fismasystemid, alt.functionoptionid
			  FROM scores s
			  JOIN functionoptions cur ON cur.functionoptionid = s.functionoptionid
			  JOIN functionoptions alt ON alt.functionid = cur.functionid AND alt.score <> cur.score
			 WHERE s.datacallid = $1
			 ORDER BY s.scoreid, alt.functionoptionid LIMIT 1`, dc).Scan(&edited.ScoreID, &edited.FismaSystemID, &edited.FunctionOptionID))
		edited.DataCallID = dc
		_, err := edited.Save(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, `UPDATE scores SET status = 'done' WHERE datacallid = $1 AND scoreid % 3 = 0`, dc)
		require.NoError(t, err)

		before, beforeAggregate, beforeProgress := readCallState(t, ctx, conn, dc), aggregate(dc), progress(dc)
		var historyRows, revisionsBefore int64
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT COUNT(DISTINCT scoreid), COUNT(*) FROM score_revisions WHERE datacallid = $1`, dc).Scan(&historyRows, &revisionsBefore))
		require.Positive(t, historyRows)

		res, err := RepinDataCall(ctx, dc, identical)
		require.NoError(t, err)
		assert.EqualValues(t, len(before.rows), res.Translated)
		assert.Zero(t, res.Dropped)

		after := readCallState(t, ctx, conn, dc)
		assert.Equal(t, []int32{identical}, after.versions, "every answer now names the target version's option")
		assert.Equal(t, before.rows, after.rows, "same lineage and score; notes, status and last_updated_at byte-identical")
		assert.Equal(t, beforeAggregate, aggregate(dc), "every computed score is identical")
		assert.Equal(t, beforeProgress, progress(dc), "progress, lastupdatedat included, is identical")

		var translates, translateVersion int64
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT COUNT(*), MIN(versionid) FROM score_revisions WHERE datacallid = $1 AND kind = 'translate'`, dc).Scan(&translates, &translateVersion))
		assert.Equal(t, historyRows, translates, "one translate revision per answer that had history, none for the rest")
		assert.EqualValues(t, identical, translateVersion, "translate revisions record the target version")

		stored := mustFindScore(t, edited.ScoreID)
		history, err := FindScoreRevisions(ctx, stored, writePolicy)
		require.NoError(t, err)
		require.Equal(t, "translate", history.Head.Kind)
		assert.Equal(t, edited.FunctionOptionID, history.Revisions[0].Prev.FunctionOptionID, "the translate records the old option")
		assert.Equal(t, stored.FunctionOptionID, history.Revisions[0].New.FunctionOptionID)
		for _, r := range history.Revisions {
			assert.NotNil(t, r.New.OptionName, "earlier revisions still read with labels")
		}
		head := history.Head.RevisionID
		_, err = UndoScoreRevision(ctx, stored, &head)
		var invalid *InvalidInputError
		require.ErrorAs(t, err, &invalid)
		assert.Equal(t, reasonIsTranslate, invalid.Data()["expected_head_revisionid"])

		var events int
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT COUNT(*) FROM events WHERE resource = 'datacalls' AND (payload->>'datacallid')::int = $1
			   AND (payload->>'to_versionid')::int = $2`, dc, identical).Scan(&events))
		assert.Equal(t, 1, events, "the re-pin is audited")
	})

	t.Run("GivenAnswersThatDoNotTranslateAre409", func(t *testing.T) {
		dc := openCallWithAnswers(t, ctx, conn, "repin_conflict", "2100-01-01")
		_, err := conn.Exec(ctx, `UPDATE scores SET status = 'done' WHERE scoreid = (SELECT MIN(scoreid) FROM scores WHERE datacallid = $1)`, dc)
		require.NoError(t, err)
		before := readCallState(t, ctx, conn, dc)

		_, err = RepinDataCall(ctx, dc, missing)
		assert.ErrorIs(t, err, ErrVersionNotTranslatable)
		assert.Equal(t, before, readCallState(t, ctx, conn, dc), "nothing changed")
		assert.EqualValues(t, 1, pin(dc))
	})

	t.Run("ASystemMarkedCompleteIsTouched", func(t *testing.T) {
		dc := openCallWithAnswers(t, ctx, conn, "repin_complete", "2100-01-01")
		_, err := conn.Exec(ctx, `
			INSERT INTO datacalls_fismasystems (datacallid, fismasystemid)
			SELECT $1, MIN(fismasystemid) FROM scores WHERE datacallid = $1`, dc)
		require.NoError(t, err)
		before := readCallState(t, ctx, conn, dc)

		_, err = RepinDataCall(ctx, dc, missing)
		assert.ErrorIs(t, err, ErrVersionNotTranslatable)
		assert.Equal(t, before, readCallState(t, ctx, conn, dc), "nothing changed")
	})

	t.Run("CurrentPinIsANoOpAfterItsArchived", func(t *testing.T) {
		dc := openCallWithAnswers(t, ctx, conn, "repin_archived", "2100-01-01")
		_, err := ArchiveQuestionnaireVersion(ctx, 1)
		require.NoError(t, err)
		t.Cleanup(func() {
			deleteFixtureRow(t, `UPDATE questionnaire_versions SET state = 'published' WHERE versionid = $1`, 1)
		})
		res, err := RepinDataCall(ctx, dc, 1)
		require.NoError(t, err)
		assert.EqualValues(t, 1, res.QuestionnaireVersionID)
	})

	t.Run("Refusals", func(t *testing.T) {
		dc := openCallWithAnswers(t, ctx, conn, "repin_refused", "2100-01-01")
		draft, _, _ := minimalDraft(t, ctx, conn, "AWS")
		_, err := RepinDataCall(ctx, dc, draft)
		requireInvalidOn(t, err, "questionnaire_versionid", draft)
		_, err = RepinDataCall(ctx, dc, 0)
		requireInvalidOn(t, err, "questionnaire_versionid")
		_, err = RepinDataCall(ctx, 999999, identical)
		assert.ErrorIs(t, err, ErrNoData)

		closed := openCallWithAnswers(t, ctx, conn, "repin_closed", "2000-01-01")
		_, err = RepinDataCall(ctx, closed, identical)
		assert.ErrorIs(t, err, ErrPastDeadline)

		res, err := RepinDataCall(ctx, dc, 1)
		require.NoError(t, err, "the version it already has is a no-op")
		assert.Zero(t, res.Translated)
	})
}

// A re-pin takes the data call before any score, so a save in flight (which
// holds the data call FOR SHARE, #395) finishes first and the two cannot
// deadlock.
func TestDataCallRepinWaitsForScoreWritesIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)
	identical := forkIdenticalVersion(t, conn)
	dc := openCallWithAnswers(t, ctx, conn, "repin_wait", "2100-01-01")

	save, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer save.Release()
	saveTx, err := save.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = saveTx.Rollback(ctx) }()
	_, err = saveTx.Exec(ctx, `SELECT 1 FROM datacalls WHERE datacallid = $1 FOR SHARE`, dc)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, err := RepinDataCall(ctx, dc, identical)
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("the re-pin finished under a save in flight: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	require.NoError(t, saveTx.Commit(ctx))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("the re-pin never finished after the save committed")
	}
}

// A system marking the call complete while a re-pin starts: the re-pin waits
// for it through the completion's FK check, then counts the system as touched
// and refuses, rather than dropping its answers.
func TestDataCallRepinWaitsForCompletionIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)
	dc := openCallWithAnswers(t, ctx, conn, "repin_wait_complete", "2100-01-01")
	root, score, _ := mostAnswered(t, ctx, conn, dc)
	missing := forkIdenticalVersion(t, conn)
	withoutOption(t, ctx, conn, missing, root, score)
	var system int32
	require.NoError(t, conn.QueryRow(ctx, `SELECT MIN(fismasystemid) FROM scores WHERE datacallid = $1`, dc).Scan(&system))

	complete, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer complete.Release()
	completeTx, err := complete.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = completeTx.Rollback(ctx) }()
	// What DataCallFismaSystem.Save runs, held open.
	_, err = completeTx.Exec(ctx, `INSERT INTO datacalls_fismasystems (datacallid, fismasystemid) VALUES ($1, $2)`, dc, system)
	require.NoError(t, err)
	var completePID int32
	require.NoError(t, completeTx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&completePID))

	done := make(chan error, 1)
	go func() {
		_, err := RepinDataCall(ctx, dc, missing)
		done <- err
	}()
	// Wait until the re-pin is blocked on the completion, or has finished
	// without waiting (the defect), before letting the completion commit.
	require.Eventually(t, func() bool {
		select {
		case err := <-done:
			done <- err
			return true
		default:
		}
		var blocked bool
		_ = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity
			 WHERE $1 = ANY(pg_blocking_pids(pid)))`, completePID).Scan(&blocked)
		return blocked
	}, 10*time.Second, 20*time.Millisecond)
	require.NoError(t, completeTx.Commit(ctx))

	select {
	case err := <-done:
		assert.ErrorIs(t, err, ErrVersionNotTranslatable, "the completion is counted, so its answers are not dropped")
	case <-time.After(10 * time.Second):
		t.Fatal("the re-pin never finished after the completion committed")
	}
}

func TestRolloverTranslatesAcrossVersionsIntegration(t *testing.T) {
	ctx, conn := versionTestSetup(t)
	// Each subtest makes its own source call: a fork's cleanup purges every
	// integration_test_ call when its subtest ends.
	var prev int32
	var source callState
	newSource := func(t *testing.T) {
		prev = openCallWithAnswers(t, ctx, conn, "rollover_prev", "2100-01-01")
		// Answers given in the source cycle, so the copy's reset is visible.
		_, err := conn.Exec(ctx, `UPDATE scores SET status = 'done', last_updated_at = NOW() WHERE datacallid = $1`, prev)
		require.NoError(t, err)
		source = readCallState(t, ctx, conn, prev)
	}

	rollover := func(t *testing.T, version int32) (int32, string) {
		t.Helper()
		var dc *DataCall
		var err error
		logged := captureRolloverLog(t, func() {
			dc, err = (&DataCall{
				DataCall:               fmt.Sprintf("%srollover_new_%d", integrationTestPrefix, time.Now().UnixNano()),
				Deadline:               time.Date(2101, 1, 1, 0, 0, 0, 0, time.UTC),
				QuestionnaireVersionID: version,
			}).Save(ctx)
		})
		require.NoError(t, err)
		t.Cleanup(func() { deleteFixtureRow(t, `DELETE FROM datacalls WHERE datacallid = $1`, dc.DataCallID) })
		assert.NotContains(t, logged, "ROLLOVER_ANOMALY")
		var carried int64
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT COUNT(*) FROM scores WHERE datacallid = $1 AND (status <> 'not_started' OR last_updated_at IS NOT NULL)`,
			dc.DataCallID).Scan(&carried))
		assert.Zero(t, carried, "carried rows reset to not_started with no last_updated_at")
		return dc.DataCallID, logged
	}

	t.Run("SameVersion", func(t *testing.T) {
		newSource(t)
		dc, logged := rollover(t, 1)
		assert.NotContains(t, logged, "VERSION_TRANSLATION_DROPPED")
		st := readCallState(t, ctx, conn, dc)
		assert.Equal(t, []int32{1}, st.versions)
		assert.Len(t, st.rows, len(source.rows))
	})

	t.Run("RemovedOptionIsDroppedAndCounted", func(t *testing.T) {
		newSource(t)
		root, score, lost := mostAnswered(t, ctx, conn, prev)
		v := forkIdenticalVersion(t, conn)
		withoutOption(t, ctx, conn, v, root, score)
		dc, logged := rollover(t, v)
		assert.Contains(t, logged, fmt.Sprintf("VERSION_TRANSLATION_DROPPED datacall=%d from_version=1 to_version=%d dropped=%d", dc, v, lost))
		st := readCallState(t, ctx, conn, dc)
		assert.Equal(t, []int32{v}, st.versions)
		assert.Len(t, st.rows, len(source.rows)-int(lost))
	})

	t.Run("RemovedQuestionIsDroppedAndCounted", func(t *testing.T) {
		newSource(t)
		root, _, _ := mostAnswered(t, ctx, conn, prev)
		v := forkIdenticalVersion(t, conn)
		var lost int64
		require.NoError(t, conn.QueryRow(ctx, `
			SELECT COUNT(*) FROM scores s JOIN functions f ON f.functionid = s.functionid
			  JOIN questions q ON q.questionid = f.questionid
			 WHERE s.datacallid = $1
			   AND q.root_questionid = (SELECT q2.root_questionid FROM functions f2 JOIN questions q2 ON q2.questionid = f2.questionid
			                             WHERE f2.root_functionid = $2 AND f2.versionid = 1)`, prev, root).Scan(&lost))
		for _, stmt := range []string{
			`DELETE FROM functions WHERE versionid = $1 AND questionid IN (SELECT questionid FROM questions WHERE versionid = $1
			    AND root_questionid = (SELECT q2.root_questionid FROM functions f2 JOIN questions q2 ON q2.questionid = f2.questionid
			                            WHERE f2.root_functionid = $2 AND f2.versionid = 1))`,
			`DELETE FROM questions WHERE versionid = $1 AND root_questionid = (SELECT q2.root_questionid FROM functions f2
			    JOIN questions q2 ON q2.questionid = f2.questionid WHERE f2.root_functionid = $2 AND f2.versionid = 1)`,
		} {
			_, err := conn.Exec(ctx, stmt, v, root)
			require.NoError(t, err)
		}
		dc, logged := rollover(t, v)
		assert.Contains(t, logged, fmt.Sprintf("dropped=%d", lost))
		assert.Len(t, readCallState(t, ctx, conn, dc).rows, len(source.rows)-int(lost))
	})

	t.Run("AddedQuestionNeedsNoAnswer", func(t *testing.T) {
		newSource(t)
		same, _ := rollover(t, 1)
		v := forkIdenticalVersion(t, conn)
		// A new question for every edition the call's systems are on.
		_, err := conn.Exec(ctx, `
			WITH q AS (INSERT INTO questions (question, notesprompt, pillarid, versionid)
			           VALUES ('added in this version', 'p', 1, $1) RETURNING questionid),
			     f AS (INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid)
			           SELECT 'added', 'd', e.env, q.questionid, 1
			             FROM q, (SELECT DISTINCT datacenterenvironment AS env FROM functions WHERE versionid = $1) e
			           RETURNING functionid)
			INSERT INTO functionoptions (functionid, score, optionname, description)
			SELECT f.functionid, s, 'o' || s, 'd' FROM f, generate_series(1, 4) s`, v)
		require.NoError(t, err)
		dc, _ := rollover(t, v)
		before, err := FindScoreProgress(ctx, FindScoreProgressInput{DataCallID: &same})
		require.NoError(t, err)
		after, err := FindScoreProgress(ctx, FindScoreProgressInput{DataCallID: &dc})
		require.NoError(t, err)
		require.Equal(t, len(before), len(after))
		bySystem := map[int32]*ScoreProgress{}
		for _, p := range before {
			bySystem[p.FismaSystemID] = p
		}
		grew := 0
		for _, p := range after {
			b := bySystem[p.FismaSystemID]
			require.NotNil(t, b)
			assert.Equal(t, b.QuestionsAnswered, p.QuestionsAnswered, "nothing answers the new question")
			if p.QuestionsExpected == b.QuestionsExpected+1 {
				grew++
			} else {
				assert.Equal(t, b.QuestionsExpected, p.QuestionsExpected)
			}
		}
		assert.Positive(t, grew, "the new question is expected of the systems on its editions")
	})
}
