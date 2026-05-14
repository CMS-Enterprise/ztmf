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
-- The ON CONFLICT clause makes re-running the migration idempotent in case
-- a row was inserted by the application layer between migration steps.
INSERT INTO public.users_opdivs (userid, opdiv_id, granted_by)
SELECT u.userid,
       (SELECT opdiv_id FROM public.opdivs WHERE code = 'CMS'),
       NULL
  FROM public.users u
 WHERE NOT EXISTS (
     SELECT 1 FROM public.users_opdivs uo
      WHERE uo.userid = u.userid
        AND uo.opdiv_id = (SELECT opdiv_id FROM public.opdivs WHERE code = 'CMS')
 )
ON CONFLICT DO NOTHING;
        `,
		`
DELETE FROM public.users_opdivs
 WHERE opdiv_id = (SELECT opdiv_id FROM public.opdivs WHERE code = 'CMS');
        `)
}
