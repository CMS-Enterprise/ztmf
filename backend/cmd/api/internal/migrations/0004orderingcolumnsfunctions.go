package migrations

func init() {
	appendMigration(
		"ordering columns",
		`ALTER TABLE IF EXISTS public.functions ADD COLUMN IF NOT EXISTS "ordr" smallint DEFAULT 0;
		`,
		`ALTER TABLE IF EXISTS public.functions DROP COLUMN IF EXISTS ordr;
		`)
}
