package migrations

func init() {
	getMigrator().AppendMigration(
		"add per-answer status to scores (state machine replacing events-derived progress)",
		`
-- Persist each answer's review state for its data call as a first-class column
-- (ztmf#435). "Data Call Progress" used to infer "updated this cycle" by
-- lateral-joining the events audit table, which is fire-and-forget,
-- non-transactional, and context-gated - a dropped event silently under-counted
-- progress. status makes the fact explicit and is written in the SAME statement
-- as the answer, so it can never disagree with the row it describes.
--
-- Two states (the third, per-answer "complete", has no trigger in the app today
-- - completion is tracked at the system level via datacalls_fismasystems - so it
-- is intentionally out of scope; see ztmf#435 open questions):
--   not_started - carried forward by copyPreviousScores, untouched this cycle
--                 (the state that used to be inferred from "no event exists")
--   done        - genuinely saved this cycle (what an edit event used to proxy)
--
-- varchar + CHECK rather than a native ENUM, matching the fismasystems
-- target_maturity_tier convention (0046): easier to widen later without an
-- ALTER TYPE dance if the third state is ever introduced.
--
-- DEFAULT 'not_started' is a backstop only; all three score-mutating paths set
-- status explicitly (scores.Save INSERT/UPDATE -> 'done', copyPreviousScores ->
-- 'not_started').
ALTER TABLE public.scores
  ADD COLUMN IF NOT EXISTS status varchar(20) NOT NULL DEFAULT 'not_started'
    CONSTRAINT scores_status_check CHECK (status IN ('not_started', 'done'));

-- Backfill mirrors the exact derivation the events lateral performed at read
-- time, so dashboards do not move at cutover: a score row with >= 1 recorded
-- edit event was "updated" (-> done); a carried-over row with none stays
-- not_started. Uses the same resource + payload->>'scoreid' predicate the
-- progress query used, which the events_score_audit_idx partial index (0037)
-- serves directly, so the backfill is index-assisted rather than a seq scan
-- per row.
--
-- Only genuine in-app edits (action 'created'/'updated') count as done. Data
-- loaded outside the app is attributed with 'imported' provenance events so it
-- carries a who/when, but an import is not a human answering this cycle - those
-- events must NOT flip the row to done, so they are excluded here.
UPDATE public.scores s
SET status = 'done'
WHERE EXISTS (
    SELECT 1
    FROM public.events e
    WHERE e.resource = 'public.scores'
      AND e.action IN ('created', 'updated')
      AND (e.payload->>'scoreid')::int = s.scoreid
);
		`,
		`
ALTER TABLE public.scores
  DROP COLUMN IF EXISTS status;
		`)
}
