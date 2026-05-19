package migrations

func init() {
	getMigrator().AppendMigration(
		"add fismasystems.opdiv_id (nullable) with FK to opdivs",
		`
ALTER TABLE IF EXISTS public.fismasystems
    ADD COLUMN IF NOT EXISTS opdiv_id INTEGER;

ALTER TABLE IF EXISTS public.fismasystems
    DROP CONSTRAINT IF EXISTS fk_fismasystems_opdiv;

ALTER TABLE IF EXISTS public.fismasystems
    ADD CONSTRAINT fk_fismasystems_opdiv
    FOREIGN KEY (opdiv_id)
    REFERENCES public.opdivs(opdiv_id)
    ON DELETE RESTRICT;

COMMENT ON COLUMN public.fismasystems.opdiv_id IS 'Owning OpDiv. Backfilled to CMS for all pre-multi-tenant rows. Becomes NOT NULL once backfill confirmed (migration 0029).';
        `,
		`
ALTER TABLE IF EXISTS public.fismasystems
    DROP CONSTRAINT IF EXISTS fk_fismasystems_opdiv,
    DROP COLUMN IF EXISTS opdiv_id;
        `)
}
