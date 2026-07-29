package migrations

func init() {
	appendMigration(
		"fismasystems decommission",
		`
ALTER TABLE IF EXISTS public.fismasystems
    ADD COLUMN decommissioned BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN decommissioned_date TIMESTAMP WITH TIME ZONE;

-- Migrate existing systems using datacenterenvironment workaround
UPDATE public.fismasystems
SET decommissioned = TRUE,
    decommissioned_date = NOW()
WHERE datacenterenvironment = 'DECOMMISSIONED';
        `,
		`
ALTER TABLE IF EXISTS public.fismasystems
    DROP COLUMN IF EXISTS decommissioned,
    DROP COLUMN IF EXISTS decommissioned_date;
        `)
}
