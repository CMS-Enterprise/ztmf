package migrations

func init() {
	getMigrator().AppendMigration(
		"add opdivs.insights_enabled flag and enable it for CMS",
		`
-- System enrichment (ZTMF Insights) is only available for OpDivs that have an
-- insights pipeline feeding system_enrichment. Today that is CMS only (the only
-- OpDiv with security-insight logs). Rather than hardcode code='CMS' in the read
-- path, gate on a per-OpDiv capability flag so enabling another OpDiv later is a
-- single UPDATE (no code change / redeploy).
--
-- Defaults FALSE so no OpDiv surfaces enrichment until explicitly enabled; the
-- backfill turns it on for the active CMS row. NOT NULL keeps the read-path join
-- predicate simple (no three-valued logic). IF NOT EXISTS for idempotent retry.

ALTER TABLE public.opdivs
    ADD COLUMN IF NOT EXISTS insights_enabled BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE public.opdivs
    SET insights_enabled = TRUE
    WHERE code = 'CMS' AND active = TRUE;
        `,
		`
ALTER TABLE public.opdivs DROP COLUMN IF EXISTS insights_enabled;
        `)
}
