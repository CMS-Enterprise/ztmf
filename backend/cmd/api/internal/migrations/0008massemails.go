package migrations

func init() {
	getMigrator().AppendMigration(
		// this table will only ever hold 1 row which will be updated when emails are sent
		// change history will then be captured in the events log
		"massemails table",
		`CREATE TABLE IF NOT EXISTS public.massemails
(
	massemailid SMALLINT PRIMARY KEY DEFAULT 1 CHECK (massemailid=1),
	datesent TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
	subject varchar(100),
	body varchar(2000)
);

INSERT INTO public.massemails (subject, body) VALUES ('-','-');
		`,
		`DROP TABLE IF EXISTS public.massemails;`)
}
