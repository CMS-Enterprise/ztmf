package migrations

func init() {
	getMigrator().AppendMigration(
		"enhance fismasystems decommission",
		`
-- Drop incorrectly typed column if exists from failed migration
ALTER TABLE IF EXISTS public.fismasystems
    DROP COLUMN IF EXISTS decommissioned_by CASCADE;

-- Add decommission audit columns
ALTER TABLE IF EXISTS public.fismasystems
    ADD COLUMN decommissioned_by UUID,
    ADD COLUMN decommissioned_notes TEXT;

-- Add foreign key to users table
ALTER TABLE IF EXISTS public.fismasystems
    ADD CONSTRAINT fk_decommissioned_by
    FOREIGN KEY (decommissioned_by)
    REFERENCES users(userid)
    ON DELETE SET NULL;

-- Add index for querying by decommissioned user
CREATE INDEX IF NOT EXISTS idx_fismasystems_decommissioned_by
    ON public.fismasystems(decommissioned_by)
    WHERE decommissioned = TRUE;

-- Add comments for documentation
COMMENT ON COLUMN public.fismasystems.decommissioned_by IS 'User ID who decommissioned the system';
COMMENT ON COLUMN public.fismasystems.decommissioned_notes IS 'Reason or notes for decommissioning';
        `,
        `
ALTER TABLE IF EXISTS public.fismasystems
    DROP CONSTRAINT IF EXISTS fk_decommissioned_by,
    DROP COLUMN IF EXISTS decommissioned_by;

DROP INDEX IF EXISTS idx_fismasystems_decommissioned_by;
        `)
}
