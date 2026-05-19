package migrations

func init() {
	getMigrator().AppendMigration(
		"add users.identity_provider (nullable) per ztmf#266",
		`
ALTER TABLE IF EXISTS public.users
    ADD COLUMN IF NOT EXISTS identity_provider VARCHAR(50);

COMMENT ON COLUMN public.users.identity_provider IS 'Which IdP authenticates this user (okta for CMS, entra for HHS/OpDivs). Managed by the application, not user-editable. Backfilled to okta for pre-multi-IdP rows (migration 0031).';
        `,
		`
ALTER TABLE IF EXISTS public.users
    DROP COLUMN IF EXISTS identity_provider;
        `)
}
