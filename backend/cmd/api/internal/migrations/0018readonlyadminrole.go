package migrations

func init() {
	getMigrator().AppendMigration(
		"widen users role column for READONLY_ADMIN",
		`ALTER TABLE public.users ALTER COLUMN role TYPE varchar(20);`,
		`ALTER TABLE public.users ALTER COLUMN role TYPE varchar(5);`)
}
