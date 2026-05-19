package migrations

func init() {
	getMigrator().AppendMigration(
		"widen users.role column to fit new OpDiv role names",
		`
-- The new role constants (OPDIV_READONLY_ADMIN at 20 chars, HHS_READONLY_ADMIN
-- at 18 chars) push right up against the existing VARCHAR(20) cap. Widen with
-- headroom so future additions do not require another column-type change.
ALTER TABLE IF EXISTS public.users
    ALTER COLUMN role TYPE VARCHAR(30);
        `,
		`
-- Shrink only when no current value exceeds 20 characters. Postgres will fail
-- the cast if a longer value exists, surfacing the rollback risk explicitly.
ALTER TABLE IF EXISTS public.users
    ALTER COLUMN role TYPE VARCHAR(20);
        `)
}
