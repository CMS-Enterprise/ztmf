package migrations

func init() {
	getMigrator().AppendMigration(
		"scores datacallid constraint",
		`ALTER TABLE public.scores DROP CONSTRAINT IF EXISTS scores_datacallid_fkey;
		ALTER TABLE public.scores ADD CONSTRAINT scores_datacallid_fkey FOREIGN KEY (datacallid) REFERENCES datacalls(datacallid) ON DELETE CASCADE;
		`,
		``)
}
