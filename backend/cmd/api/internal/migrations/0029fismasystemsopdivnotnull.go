package migrations

func init() {
	getMigrator().AppendMigration(
		"enforce NOT NULL on fismasystems.opdiv_id and add index",
		`
ALTER TABLE IF EXISTS public.fismasystems
    ALTER COLUMN opdiv_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_fismasystems_opdiv_id
    ON public.fismasystems(opdiv_id);
        `,
		`
DROP INDEX IF EXISTS public.idx_fismasystems_opdiv_id;

ALTER TABLE IF EXISTS public.fismasystems
    ALTER COLUMN opdiv_id DROP NOT NULL;
        `)
}
