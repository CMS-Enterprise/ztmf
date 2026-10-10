package migrations

func init() {
	appendMigration(
		"add scores(datacallid, fismasystemid) index for the score aggregate",
		`
-- ztmf-misc#426. /scores/aggregate filters scores by datacallid alone, and the only
-- composite index (0061) leads with fismasystemid, so every call read every
-- cycle's rows. On a prod-scale copy this cut the aggregate's
-- page reads by a quarter and stops them growing with each new data call.
--
-- Ordered after the 0064 backfill so its row rewrites don't bloat this index.
-- Not CONCURRENTLY, for the reason in 0061.
SET LOCAL lock_timeout = '10s';

CREATE INDEX IF NOT EXISTS scores_datacall_fismasystem_idx
    ON public.scores (datacallid, fismasystemid);
`,
		`DROP INDEX IF EXISTS scores_datacall_fismasystem_idx;`,
	)
}
