package migrations

func init() {
	getMigrator().AppendMigration(
		"constrain users.access_expires_at to SYSTEM_DELEGATE rows only",
		`
-- Defense-in-depth for the System Delegate expiry invariant: only a
-- SYSTEM_DELEGATE row may carry a non-null access_expires_at. Today every write
-- path upholds this in application code (scores.Save/SaveUser blank it, re-role
-- clears it), but a future write path that forgets would silently plant an
-- expiry on a non-delegate. This CHECK makes the DB the backstop.
--
-- Safe on the populated table: 0050 left every existing row NULL and only the
-- delegate add/renew paths set it, so no current row violates the constraint.
-- Guarded by a DO block because Postgres has no ADD CONSTRAINT IF NOT EXISTS,
-- so the migration stays idempotent on retry.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_delegate_expiry_ck'
          AND conrelid = 'public.users'::regclass
    ) THEN
        ALTER TABLE public.users
            ADD CONSTRAINT users_delegate_expiry_ck
            CHECK (role = 'SYSTEM_DELEGATE' OR access_expires_at IS NULL);
    END IF;
END $$;
        `,
		`
ALTER TABLE public.users DROP CONSTRAINT IF EXISTS users_delegate_expiry_ck;
        `)
}
