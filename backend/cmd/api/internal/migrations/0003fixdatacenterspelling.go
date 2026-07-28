package migrations

func init() {
	appendMigration(
		"fix mispelling",
		`UPDATE public.fismasystems SET datacenterenvironment='DECOMMISSIONED' where datacenterenvironment='DECOMISSIONED';`,
		"")
}
