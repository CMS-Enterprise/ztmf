package migrations

func init() {
	getMigrator().AppendMigration(
		"seed users_opdivs with CMS grant for every existing user",
		`
-- Day 1 of the multi-OpDiv schema: every pre-existing user is a CMS user.
-- Grant them CMS membership so they retain the access they had before this
-- migration. New users provisioned through the admin panel or via the HHS
-- onboarding workbook importer get explicit grants at create time.
--
-- granted_by is left NULL because there is no human grantor for this seed.
-- The down migration uses this NULL marker to delete only seed rows, leaving
-- human-created grants intact.
--
-- The ON CONFLICT clause makes re-running the migration idempotent in case
-- a row was inserted by the application layer between migration steps.
DO $$
DECLARE
    cms_id integer;
BEGIN
    SELECT opdiv_id INTO cms_id
      FROM public.opdivs
     WHERE code = 'CMS' AND active = TRUE
     LIMIT 1;

    IF cms_id IS NULL THEN
        RAISE EXCEPTION 'cannot seed users_opdivs: no active CMS row in opdivs (check migration 0026 seed)';
    END IF;

    INSERT INTO public.users_opdivs (userid, opdiv_id, granted_by)
    SELECT u.userid, cms_id, NULL
      FROM public.users u
     WHERE NOT EXISTS (
         SELECT 1 FROM public.users_opdivs uo
          WHERE uo.userid = u.userid
            AND uo.opdiv_id = cms_id
     )
    ON CONFLICT DO NOTHING;
END $$;
        `,
		`
-- Only delete seed rows (granted_by IS NULL marker) so any human-created CMS
-- grants survive a rollback from a later stage.
DELETE FROM public.users_opdivs
 WHERE opdiv_id = (SELECT opdiv_id FROM public.opdivs WHERE code = 'CMS')
   AND granted_by IS NULL;
        `)
}
