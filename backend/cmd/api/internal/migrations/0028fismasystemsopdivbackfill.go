package migrations

func init() {
	getMigrator().AppendMigration(
		"backfill fismasystems.opdiv_id to CMS",
		`
-- Every fismasystem row predates the multi-OpDiv schema and belongs to CMS by
-- definition. CFACTS component_acronym values are CMS-internal organizational
-- codes (OEDA, CCIIO, ...), not OpDiv codes, so we cannot derive opdiv_id from
-- the existing CFACTS join. Default everything to CMS unconditionally; HHS
-- OpDiv data arrives via the onboarding workbook importer in a later phase.
--
-- Fail loudly if the CMS row is somehow missing or inactive at this point
-- (e.g. the 0026 seed was edited out and migrations ran on a fresh DB).
-- Otherwise the UPDATE would set opdiv_id to NULL and migration 0029 would
-- then fail with a less obvious NOT NULL violation.
DO $$
DECLARE
    cms_id integer;
BEGIN
    SELECT opdiv_id INTO cms_id
      FROM public.opdivs
     WHERE code = 'CMS' AND active = TRUE
     LIMIT 1;

    IF cms_id IS NULL THEN
        RAISE EXCEPTION 'cannot backfill fismasystems.opdiv_id: no active CMS row in opdivs (check migration 0026 seed)';
    END IF;

    UPDATE public.fismasystems
       SET opdiv_id = cms_id
     WHERE opdiv_id IS NULL;
END $$;
        `,
		`
-- No-op on rollback. Re-running the up migration is idempotent (the WHERE
-- clause is satisfied only by rows still missing an opdiv_id), and rolling back
-- the column-add migration (0027) drops the column entirely.
SELECT 1;
        `)
}
