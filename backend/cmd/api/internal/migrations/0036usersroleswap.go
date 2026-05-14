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
-- Elizabeth Schweinsberg (CMS Zero Trust program lead) is promoted to
-- HHS_ADMIN instead of OWNER because she is the day-1 HHS-tier admin per the
-- multi-OpDiv plan. We match by name and email pattern rather than a hardcoded
-- email so the migration is environment-agnostic; if no row matches in a given
-- environment (e.g. fresh local DBs), this step is a no-op for her and she
-- gets provisioned later through the admin panel.
--
-- Order matters: Elizabeth's HHS_ADMIN assignment runs first so the bulk
-- ADMIN -> OWNER update below skips her by virtue of her role no longer being
-- ADMIN.

UPDATE public.users
   SET role = 'HHS_ADMIN'
 WHERE role = 'ADMIN'
   AND (
        LOWER(email)    LIKE 'elizabeth.schweinsberg%@%'
     OR LOWER(fullname) LIKE '%elizabeth%schweinsberg%'
       );

UPDATE public.users
   SET role = 'OWNER'
 WHERE role = 'ADMIN';

UPDATE public.users
   SET role = 'HHS_READONLY_ADMIN'
 WHERE role = 'READONLY_ADMIN';
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
