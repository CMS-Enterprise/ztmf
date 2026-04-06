package migrations

func init() {
	getMigrator().AppendMigration(
		"add auth_methods and fips_impact_level to cfacts_systems",
		`
ALTER TABLE public.cfacts_systems ADD COLUMN IF NOT EXISTS auth_methods TEXT;
ALTER TABLE public.cfacts_systems ADD COLUMN IF NOT EXISTS fips_impact_level VARCHAR(20);
		`,
		`
ALTER TABLE public.cfacts_systems DROP COLUMN IF EXISTS auth_methods;
ALTER TABLE public.cfacts_systems DROP COLUMN IF EXISTS fips_impact_level;
		`)
}
