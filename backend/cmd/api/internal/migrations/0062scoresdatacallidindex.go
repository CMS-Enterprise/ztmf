package migrations

func init() {
	appendMigration(
		"scores datacallid index for dashboard aggregate and progress",
		`
-- ztmf#607. /scores/aggregate builds scored_pairs as DISTINCT
-- (fismasystemid, datacallid) FROM scores with no leading index on
-- datacallid, so every data call re-read the whole table. /scores/progress
-- joins scores ON fismasystemid AND datacallid = $n for every visible
-- system. Lead with datacallid (the per-request filter) then system.
SET LOCAL lock_timeout = '10s';

CREATE INDEX IF NOT EXISTS scores_datacallid_fismasystemid_idx
    ON public.scores (datacallid, fismasystemid);

COMMENT ON INDEX public.scores_datacallid_fismasystemid_idx IS
  'Per data call: dashboard aggregate scored_pairs and score progress (ztmf#607). The existing unique index leads with fismasystemid and cannot serve a datacallid-first scan.';
		`,
		`DROP INDEX IF EXISTS public.scores_datacallid_fismasystemid_idx;`)
}
