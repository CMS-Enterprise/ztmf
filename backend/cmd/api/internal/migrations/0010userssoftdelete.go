package migrations

func init() {
	appendMigration(
		// add soft delete capability
		"users table role type",
		`ALTER TABLE IF EXISTS public.users ADD COLUMN deleted boolean NOT NULL DEFAULT FALSE;`,
		`ALTER TABLE IF EXISTS public.users DROP COLUMN IF EXISTS deleted;`)
}
