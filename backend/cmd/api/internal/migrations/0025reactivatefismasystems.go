package migrations

func init() {
	getMigrator().AppendMigration(
		"add fismasystems reactivation audit columns",
		`
-- Drop incorrectly typed columns if they exist from a failed prior migration attempt.
-- CASCADE on every column for consistency, even though only reactivated_by has a
-- dependent FK; safer if a future migration adds dependent objects on the others.
ALTER TABLE IF EXISTS public.fismasystems
    DROP COLUMN IF EXISTS reactivated_by CASCADE,
    DROP COLUMN IF EXISTS reactivated_date CASCADE,
    DROP COLUMN IF EXISTS reactivation_notes CASCADE;

-- Add reactivation audit columns
ALTER TABLE IF EXISTS public.fismasystems
    ADD COLUMN reactivated_by UUID,
    ADD COLUMN reactivated_date TIMESTAMP WITH TIME ZONE,
    ADD COLUMN reactivation_notes TEXT;

-- Foreign key to users so reactivator history survives user soft-deletes
ALTER TABLE IF EXISTS public.fismasystems
    ADD CONSTRAINT fk_reactivated_by
    FOREIGN KEY (reactivated_by)
    REFERENCES users(userid)
    ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_fismasystems_reactivated_by
    ON public.fismasystems(reactivated_by)
    WHERE reactivated_by IS NOT NULL;

COMMENT ON COLUMN public.fismasystems.reactivated_by IS 'User ID who reactivated the system after decommission';
COMMENT ON COLUMN public.fismasystems.reactivated_date IS 'Timestamp the system was reactivated';
COMMENT ON COLUMN public.fismasystems.reactivation_notes IS 'Reason or notes for reactivation';
        `,
		`
ALTER TABLE IF EXISTS public.fismasystems
    DROP CONSTRAINT IF EXISTS fk_reactivated_by,
    DROP COLUMN IF EXISTS reactivated_by,
    DROP COLUMN IF EXISTS reactivated_date,
    DROP COLUMN IF EXISTS reactivation_notes;

DROP INDEX IF EXISTS idx_fismasystems_reactivated_by;
        `)
}
