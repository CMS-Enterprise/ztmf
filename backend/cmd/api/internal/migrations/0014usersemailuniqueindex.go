package migrations

// UNIQUE *constraints* are case sensistive, but emails should be be evaluated case-insensitive
// remove the unique constraint and add a unique index instead

func init() {
	getMigrator().AppendMigration(
		"users unique index",
		`ALTER TABLE IF EXISTS public.users DROP CONSTRAINT IF EXISTS users_email_key;
		 CREATE UNIQUE INDEX email_unique_idx on users (LOWER(email));`,
		`DROP INDEX IF EXISTS public.email_unique_idx;
		 ALTER TABLE IF EXISTS public.users ADD CONSTRAINT users_email_key UNIQUE (email);`)
}
