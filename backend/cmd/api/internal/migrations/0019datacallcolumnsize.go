package migrations

func init() {
	appendMigration(
		"widen datacalls datacall column from char(9) to varchar(100)",
		`ALTER TABLE public.datacalls ALTER COLUMN datacall TYPE varchar(100);`,
		`ALTER TABLE public.datacalls ALTER COLUMN datacall TYPE character(9);`)
}
