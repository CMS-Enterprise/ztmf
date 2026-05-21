package migrations

func init() {
	getMigrator().AppendMigration(
		"add audit-read indexes on events",
		`
-- Per-resource audit fields (ztmf-ui#310) read events via a LATERAL
-- subquery filtered on resource + payload->>'scoreid'. The events
-- table previously had zero indexes; the lateral degenerates to a
-- sequential scan per score row at any non-trivial event count.
--
-- Two indexes:
--
-- 1. events_score_audit_idx: partial expression index sized to the
--    per-question-on-dashboard hot path. Scoped to resource='public.scores'
--    so it stays small as other resources adopt the Auditable pattern,
--    and ordered by createdat DESC so the LATERAL's "most recent
--    write" lookup is an index-only descent.
--
--    The 'public.scores' literal mirrors the value events.resource
--    carries today; bare-name normalization is a separate follow-up.
--    If that landfills first, this index will need a covering update.
--
-- 2. events_resource_createdat_idx: generic resource + time index
--    that supports future Auditable resources (FismaSystem, DataCall,
--    User, MassEmail) without adding a partial index per resource.
--    Useful for admin /events filters too.
--
-- CONCURRENTLY would avoid table locks but tern wraps each migration
-- in a transaction, which forbids CONCURRENTLY. Acceptable for this
-- change: events is append-only and the build window on the current
-- prod-shaped dev DB (~24k rows) is under a second.

CREATE INDEX IF NOT EXISTS events_score_audit_idx
    ON public.events ((((payload->>'scoreid')::int)), createdat DESC)
    WHERE resource = 'public.scores';

CREATE INDEX IF NOT EXISTS events_resource_createdat_idx
    ON public.events (resource, createdat DESC);
        `,
		`
DROP INDEX IF EXISTS public.events_resource_createdat_idx;
DROP INDEX IF EXISTS public.events_score_audit_idx;
        `)
}
