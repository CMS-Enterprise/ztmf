package migrations

func init() {
	getMigrator().AppendMigration(
		"add opdivs.system_delegate_enabled toggle (default off, opt-in per OpDiv)",
		`
-- The System Delegate self-service capability (ISSO#467) is enabled per OpDiv
-- via the "Add System Delegate Role" toggle on the Manage OpDivs panel. Persist
-- it as a per-OpDiv flag rather than hardcoding any OpDiv (e.g. code='CMS'), so
-- turning the capability on for an OpDiv is a single write from the panel and no
-- OpDiv is special-cased in the code path.
--
-- Defaults FALSE: the capability is opt-in, so no OpDiv can have System Delegates
-- added until an HHS admin / Owner explicitly enables it. NOT NULL keeps the add-
-- flow gate a simple boolean check (no three-valued logic). IF NOT EXISTS for
-- idempotent retry.

ALTER TABLE public.opdivs
    ADD COLUMN IF NOT EXISTS system_delegate_enabled BOOLEAN NOT NULL DEFAULT FALSE;
        `,
		`
ALTER TABLE public.opdivs DROP COLUMN IF EXISTS system_delegate_enabled;
        `)
}
