package migrations

func init() {
	getMigrator().AppendMigration(
		"fix mispelling",
		`UPDATE public.fismasystems SET datacenterenvironment='DECOMMISSIONED' where datacenterenvironment='DECOMISSIONED';`,
		"")
}
