package migrations

func init() {
	getMigrator().AppendMigration(
		"add auth_methods to cfacts_systems",
		`
ALTER TABLE public.cfacts_systems ADD COLUMN IF NOT EXISTS auth_methods TEXT;
		`,
		`
ALTER TABLE public.cfacts_systems DROP COLUMN IF EXISTS auth_methods;
		`)
}
