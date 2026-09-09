package migrations

func init() {
	appendMigration(
		"widen massemails.group column to fit every recipient group key",
		`
-- 0011 created massemails."group" as VARCHAR(5), sized for the original keys
-- (ISSO, ISSM, DCC, ALL, ADMIN). Longer keys added since (SYSTEM_DELEGATE at
-- 15 chars, READONLY_ADMIN at 14) fail the UPDATE in MassEmail.Save with
-- "value too long for type character varying(5)" before any recipient is
-- resolved. Widen to match users.role (0035) so the column is not the limit
-- on the audience selectors the API accepts.
ALTER TABLE IF EXISTS public.massemails
    ALTER COLUMN "group" TYPE VARCHAR(30);
        `,
		`
-- Shrink only when the stored value fits. Postgres fails the cast otherwise,
-- surfacing the rollback risk explicitly.
ALTER TABLE IF EXISTS public.massemails
    ALTER COLUMN "group" TYPE VARCHAR(5);
        `)
}
