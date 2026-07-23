package migrations

func init() {
	getMigrator().AppendMigration(
		"add users.access_expires_at for System Delegate expiry (nullable, null = never)",
		`
-- System Delegate accounts (ISSO#467) carry a mandatory expiration set at the
-- ISSO add flow. Enforcement is lazy and authoritative in the auth middleware
-- (the same place soft-deleted users are rejected): a row with
-- access_expires_at < now() is denied access, no scheduled job required.
--
-- Nullable with no default: regular (non-delegate) users stay NULL and never
-- expire. Only the delegate add/renew paths set it. IF NOT EXISTS for idempotent
-- retry.

ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS access_expires_at TIMESTAMP WITH TIME ZONE;
        `,
		`
ALTER TABLE public.users DROP COLUMN IF EXISTS access_expires_at;
        `)
}
