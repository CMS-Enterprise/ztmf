package migrations

func init() {
	getMigrator().AppendMigration(
		// add group column to massemails
		"massemails group column",
		`ALTER TABLE IF EXISTS public.massemails ADD COLUMN "group" varchar(5);`,
		`ALTER TABLE IF EXISTS public.massemails DROP COLUMN IF EXISTS "group";`)
}
