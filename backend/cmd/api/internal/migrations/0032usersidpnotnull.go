package migrations

func init() {
	getMigrator().AppendMigration(
		"enforce NOT NULL on users.identity_provider",
		`
ALTER TABLE IF EXISTS public.users
    ALTER COLUMN identity_provider SET NOT NULL;
        `,
		`
ALTER TABLE IF EXISTS public.users
    ALTER COLUMN identity_provider DROP NOT NULL;
        `)
}
