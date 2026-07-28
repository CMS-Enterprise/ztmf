package migrations

func init() {
	appendMigration(
		"widen users role column for READONLY_ADMIN",
		`ALTER TABLE public.users ALTER COLUMN role TYPE varchar(20);`,
		`ALTER TABLE public.users ALTER COLUMN role TYPE varchar(5);`)
}
