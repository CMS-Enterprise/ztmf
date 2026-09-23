package migrations

func init() {
	appendMigration(
		"populate functions.ordr from the canonical questionnaire order",
		`
-- functions.ordr has existed since 0004 but is 0 on every production row.
-- Migration 0056 moved the pillar and question order into the data; it stopped
-- short of the functions table, which scorediff.go documents out loud - its
-- ORDER BY already names fn.ordr and notes the column is 0 everywhere, so the
-- key contributes nothing. FindQuestionsByFismaSystem did not even select it.
--
-- With the client-side sort deleted (ztmf-misc#393) the API response IS the
-- questionnaire order, so the last unpopulated rank has to be filled in.
--
-- Ranks are copied from the owning question rather than re-transcribed from
-- ztmf-ui's PILLAR_FUNCTION_MAP. 0056 set questions.ordr by joining that map on
-- the function NAME - the stable business key across editions - so copying
-- questions.ordr reproduces those same ranks on every edition's functions row
-- without a second 40-row VALUES list to keep in agreement with the first.
--
-- functions.ordr orders the functions a single question fans out to, and the
-- catalog gives each (question, environment) pair exactly one row, so every
-- function under a question ends up tied. That is correct: the column only
-- disambiguates a question that forks, and the read paths carry a questionid
-- tiebreaker for everything else. It exists so a future versioned catalog can
-- rank a fork without another migration.
--
-- The q.ordr guard skips questions 0056 could not rank - the empire seed's
-- fictional function names - leaving them at 0 where the tiebreakers handle
-- them. The f.ordr guard makes this idempotent and, like 0056's, refuses to
-- overwrite a rank someone set deliberately.
--
-- Both are wrapped in coalesce because ordr is nullable (0004 added it DEFAULT
-- 0 but not NOT NULL). A bare f.ordr = 0 is NULL rather than true on a NULL
-- row, so a function with a NULL rank would be skipped and left unranked - the
-- exact silent mis-ordering this ticket exists to remove.
UPDATE functions f
SET ordr = q.ordr
FROM questions q
WHERE q.questionid = f.questionid
  AND coalesce(q.ordr, 0) <> 0
  AND coalesce(f.ordr, 0) = 0;
		`,
		`UPDATE functions SET ordr = 0;
		`)
}
