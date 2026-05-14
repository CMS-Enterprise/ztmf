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
UPDATE public.fismasystems
   SET opdiv_id = (SELECT opdiv_id FROM public.opdivs WHERE code = 'CMS')
 WHERE opdiv_id IS NULL;
        `,
		`
-- No-op on rollback. Re-running the up migration is idempotent (the WHERE
-- clause is satisfied only by rows still missing an opdiv_id), and rolling back
-- the column-add migration (0027) drops the column entirely.
SELECT 1;
        `)
}
