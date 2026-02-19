package migrations

func init() {
	getMigrator().AppendMigration(
		"add sdl_sync_enabled toggle to fismasystems",
		// UP: Backfill existing rows as enabled (true), then change the column
		// default to false so new systems must explicitly opt in to SDL sync.
		`ALTER TABLE public.fismasystems ADD COLUMN IF NOT EXISTS sdl_sync_enabled BOOLEAN NOT NULL DEFAULT TRUE;
		 ALTER TABLE public.fismasystems ALTER COLUMN sdl_sync_enabled SET DEFAULT FALSE;`,
		// DOWN: Remove the column
		`ALTER TABLE public.fismasystems DROP COLUMN IF EXISTS sdl_sync_enabled;`)
}
