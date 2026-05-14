package migrations

func init() {
	getMigrator().AppendMigration(
		"backfill users.identity_provider to okta",
		`
-- Every pre-multi-IdP user is a CMS user authenticating via Okta. HHS OpDiv
-- users arrive later through the onboarding workbook importer with
-- identity_provider explicitly set to 'entra' (or 'okta' for CMS contractor
-- exceptions).
UPDATE public.users
   SET identity_provider = 'okta'
 WHERE identity_provider IS NULL;
        `,
		`
-- No-op on rollback. Re-running the up is idempotent; rolling back the
-- column-add migration (0030) drops the column entirely.
SELECT 1;
        `)
}
