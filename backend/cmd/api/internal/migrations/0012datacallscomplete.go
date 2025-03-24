package migrations

func init() {
	getMigrator().AppendMigration(
		"datacalls add complete and remove email",
		`ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS "emailsubject";
		ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS "emailbody";
		ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS "emailsent";
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "complete" boolean NOT NULL DEFAULT FALSE;
		`,
		`ALTER TABLE IF EXISTS public.datacalls DROP COLUMN IF EXISTS "complete";
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailsubject" varchar(100);
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailbody" varchar(2000);
		ALTER TABLE IF EXISTS public.datacalls ADD COLUMN IF NOT EXISTS "emailsent" timestamp with time zone[];
		`)
}
