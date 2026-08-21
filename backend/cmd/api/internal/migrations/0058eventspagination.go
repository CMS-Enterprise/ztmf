package migrations

func init() {
	appendMigration(
		"events pagination: identity tiebreaker and global order index",
		`
-- The admin events page (ztmf#564) reads the audit trail ordered by
-- createdat DESC in pages. Two things the table lacks for that:
--
-- 1. A tiebreaker. events has no primary key, and bulk writers (imports,
--    seed SQL) stamp batches of rows with identical createdat values, so
--    ORDER BY createdat alone is not a total order: tied rows may swap
--    between requests, duplicating or skipping rows across page
--    boundaries. eventid only has to make the order stable, not
--    chronological, so the backfill order Postgres assigns to existing
--    rows is fine. GENERATED ALWAYS means the INSERT sites (which all
--    name their columns) keep working unchanged and cannot supply their
--    own values.
--
-- 2. An index serving the unfiltered scan. 0037's (resource, createdat)
--    and 0054's (userid, createdat) indexes each need their leading
--    column in the predicate, and the page's default view has no filter,
--    so without this index every first page is a top-N sort over the
--    whole table.
ALTER TABLE public.events ADD COLUMN IF NOT EXISTS eventid BIGINT GENERATED ALWAYS AS IDENTITY;

CREATE INDEX IF NOT EXISTS events_pagination_idx
    ON public.events (createdat DESC, eventid DESC);
`,
		`
DROP INDEX IF EXISTS public.events_pagination_idx;
ALTER TABLE public.events DROP COLUMN IF EXISTS eventid;
`,
	)
}
