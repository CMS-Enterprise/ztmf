package migrations

func init() {
	getMigrator().AppendMigration(
		"datacalls new columnds",
		`ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailsubject" varchar(100);
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailbody" varchar(2000);
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailsent" timestamp with time zone[];
		`,
		`ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS emailsubject;
		ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS emailbody;
		ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS emailsent;
		`)
}
