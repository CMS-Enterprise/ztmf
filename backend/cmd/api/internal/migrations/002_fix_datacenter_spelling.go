package migrations

func init() {
	getMigrator().AppendMigration(
		"ordering columns",
		`UPDATE public.fismasystems SET datacenterenvironment='DECOMMISSIONED' where datacenterenvironment='DECOMISSIONED';`,
		"")
}
