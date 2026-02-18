package migrations

func init() {
	getMigrator().AppendMigration(
		"add sdl_sync_enabled toggle to fismasystems",
		// UP: Add column with DEFAULT TRUE so existing rows get true,
		// then change default to FALSE for future inserts (opt-in model)
		`ALTER TABLE public.fismasystems ADD COLUMN IF NOT EXISTS sdl_sync_enabled BOOLEAN NOT NULL DEFAULT TRUE;
		 ALTER TABLE public.fismasystems ALTER COLUMN sdl_sync_enabled SET DEFAULT FALSE;`,
		// DOWN: Remove the column
		`ALTER TABLE public.fismasystems DROP COLUMN IF EXISTS sdl_sync_enabled;`)
}
