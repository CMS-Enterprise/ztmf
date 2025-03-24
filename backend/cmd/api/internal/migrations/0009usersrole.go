package migrations

func init() {
	getMigrator().AppendMigration(
		// change char to varchar to avoid padding characters
		"users table role type",
		`ALTER TABLE public.users ALTER COLUMN role TYPE varchar(5);`,
		`ALTER TABLE public.users ALTER COLUMN role TYPE char(5);`)
}
