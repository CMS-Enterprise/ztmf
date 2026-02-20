package migrations

func init() {
	getMigrator().AppendMigration(
		"add group_acronym to cfacts_systems",
		`
ALTER TABLE public.cfacts_systems ADD COLUMN IF NOT EXISTS group_acronym VARCHAR(50);
		`,
		`
ALTER TABLE public.cfacts_systems DROP COLUMN IF EXISTS group_acronym;
		`)
}
