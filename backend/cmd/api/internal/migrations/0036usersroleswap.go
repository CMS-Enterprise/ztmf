package migrations

func init() {
	getMigrator().AppendMigration(
		"swap legacy roles to multi-OpDiv role taxonomy",
		`
-- Stage B: role swap. Maps the legacy ADMIN / READONLY_ADMIN values onto the
-- new multi-OpDiv role taxonomy. ISSO and ISSM are unchanged. The predicates
-- in validations.go continue to recognize the legacy values during transition
-- so app behavior does not change until Stage C flips the controllers.
--
-- HHS_ADMIN day-1 seed list is empty. Elizabeth Schweinsberg, originally
-- considered the day-1 HHS_ADMIN, is also the ZTMF product owner, so OWNER
-- (unscoped platform tier) is the correct mapping. She falls into the bulk
-- ADMIN -> OWNER update with everyone else. Actual HHS_ADMINs are provisioned
-- through the admin panel once HHS OpDivs are onboarded.
--
-- A RAISE NOTICE at the end logs the row counts for each swap so the result
-- is easy to verify post-deploy.

DO $$
DECLARE
    n_owner    int := 0;
    n_readonly int := 0;
BEGIN
    UPDATE public.users
       SET role = 'OWNER'
     WHERE role = 'ADMIN';
    GET DIAGNOSTICS n_owner = ROW_COUNT;

    UPDATE public.users
       SET role = 'HHS_READONLY_ADMIN'
     WHERE role = 'READONLY_ADMIN';
    GET DIAGNOSTICS n_readonly = ROW_COUNT;

    RAISE NOTICE 'role swap complete: ADMIN -> OWNER=%, READONLY_ADMIN -> HHS_READONLY_ADMIN=%',
        n_owner, n_readonly;
END $$;
        `,
		`
-- Reverse swap. Note that this rollback cannot distinguish the Elizabeth
-- exception from the rest, so all HHS_ADMIN rows become ADMIN. That is the
-- correct rollback for the migration as written: before this migration, every
-- HHS_ADMIN row was an ADMIN row.

UPDATE public.users
   SET role = 'ADMIN'
 WHERE role IN ('OWNER', 'HHS_ADMIN');

UPDATE public.users
   SET role = 'READONLY_ADMIN'
 WHERE role = 'HHS_READONLY_ADMIN';
        `)
}
