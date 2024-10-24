package migrations

func init() {
	getMigrator().AppendMigration(
		"ordering columns",
		`ALTER TABLE IF EXISTS public.pillars ADD COLUMN IF NOT EXISTS "ordr" smallint DEFAULT 0;
		 ALTER TABLE IF EXISTS public.questions ADD COLUMN IF NOT EXISTS "ordr" smallint DEFAULT 0;
		`,
		`ALTER TABLE IF EXISTS public.pillars DROP COLUMN IF EXISTS ordr;
		 ALTER TABLE IF EXISTS public.questions DROP COLUMN IF EXISTS ordr;
		`)
}
