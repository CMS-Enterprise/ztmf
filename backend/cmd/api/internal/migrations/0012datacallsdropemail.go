package migrations

// new column: datacalls.completed is intended for marking a data call complete per fisma system
// every id in the array indicates

func init() {
	getMigrator().AppendMigration(
		"datacalls remove email; create datacalls_fismasystems",
		`CREATE TABLE IF NOT EXISTS public.datacalls_fismasystems
		(
			datacallid integer NOT NULL REFERENCES datacalls (datacallid) ON DELETE CASCADE,
			fismasystemid integer NOT NULL REFERENCES fismasystems (fismasystemid) ON DELETE CASCADE,
			PRIMARY KEY (datacallid, fismasystemid)
		);
		ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS "emailsubject";
		ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS "emailbody";
		ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS "emailsent";
		`,
		`
		DROP TABLE IF EXISTS "datacalls_fismasystems";
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailsubject" varchar(100);
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailbody" varchar(2000);
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailsent" timestamp with time zone[];
		`)
}
