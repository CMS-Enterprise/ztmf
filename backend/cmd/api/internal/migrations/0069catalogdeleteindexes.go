package migrations

func init() {
	appendMigration(
		"index the score references a catalog delete checks",
		`
-- ztmf-misc#396. Deleting draft options and functions fires the NO ACTION FK
-- checks from scores (functionoptionid, and the 0060 composite that leads with
-- it) and from score_revisions (functionid). Neither column was indexed, so
-- each deleted row cost a sequential scan of scores: minutes to discard one
-- fork, all under the lifecycle lock. Both tables are written in score saves.
SET LOCAL lock_timeout = '10s';

CREATE INDEX IF NOT EXISTS scores_functionoptionid_idx
    ON public.scores (functionoptionid);
CREATE INDEX IF NOT EXISTS score_revisions_functionid_idx
    ON public.score_revisions (functionid);
		`,
		`
DROP INDEX IF EXISTS public.score_revisions_functionid_idx;
DROP INDEX IF EXISTS public.scores_functionoptionid_idx;
		`)
}
