package migrations

func init() {
	getMigrator().AppendMigration(
		"Stage D: reject legacy ADMIN / READONLY_ADMIN role values",
		`
-- Stage D role cleanup. The Stage B swap (0036usersroleswap.go) already
-- mapped ADMIN -> OWNER and READONLY_ADMIN -> HHS_READONLY_ADMIN, and the app
-- no longer recognizes the legacy values (see internal/model/validations.go
-- and users.go). This guard rejects any new write that carries a legacy value,
-- as defense in depth against a stale client or a botched rollback re-seeding
-- the old strings.
--
-- Pre-flight expectation: zero rows carry a legacy value at the moment this
-- runs (SELECT count(*) FROM users WHERE role IN ('ADMIN','READONLY_ADMIN')
-- returns 0). The Stage B swap emptied them, so a plain ADD CONSTRAINT
-- validates against existing rows without a separate NOT VALID / VALIDATE
-- pass. If a stray legacy row survived, this ALTER fails loudly rather than
-- silently accepting bad data, which is the behavior we want.
ALTER TABLE public.users
    ADD CONSTRAINT users_role_no_legacy
    CHECK (role NOT IN ('ADMIN', 'READONLY_ADMIN'));
		`,
		`
-- Rollback drops the guard so the prior release (whose validation map and role
-- helpers still recognize ADMIN / READONLY_ADMIN) can write those values again
-- during the soak window. Schema-only: the legacy values live in the app's
-- validation map and helpers, which are restored by redeploying the prior
-- binary alongside this down migration.
ALTER TABLE public.users
    DROP CONSTRAINT IF EXISTS users_role_no_legacy;
		`)
}
