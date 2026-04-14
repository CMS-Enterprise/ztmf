package migrations

func init() {
	getMigrator().AppendMigration(
		"widen scores notes column from varchar(1000) to varchar(2000)",
		`ALTER TABLE public.scores ALTER COLUMN notes TYPE character varying(2000);`,
		`ALTER TABLE public.scores ALTER COLUMN notes TYPE character varying(1000);`)
}
