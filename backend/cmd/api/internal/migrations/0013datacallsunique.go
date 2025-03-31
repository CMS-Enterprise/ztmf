package migrations

// new column: datacalls.completed is intended for marking a data call complete per fisma system
// every id in the array indicates

func init() {
	getMigrator().AppendMigration(
		"datacalls unique data call column",
		`ALTER TABLE IF EXISTS public.datacalls DROP CONSTRAINT IF EXISTS datacall_key;
		ALTER TABLE IF EXISTS public.datacalls ADD CONSTRAINT datacall_key UNIQUE (datacall);
		`,
		`ALTER TABLE IF EXISTS public.datacalls DROP CONSTRAINT IF EXISTS datacall_key;`)
}
