package migrations

func init() {
	getMigrator().AppendMigration(
		"user events",
		`CREATE TABLE IF NOT EXISTS public.events
		 (
			userid uuid NOT NULL REFERENCES USERS (userid),
			action VARCHAR(30) NOT NULL, 
			resource VARCHAR(30) NOT NULL,
			createdat TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			payload JSONB NOT NULL
		 );
		`,
		`
		DROP TABLE IF EXISTS public.events CASCADE;
		`)
}
