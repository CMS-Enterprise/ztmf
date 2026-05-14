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
-- multi-OpDiv plan. We match by email first, falling back to a tight fullname
-- match, so the migration works across environments without hardcoding a
-- single literal value. The email pattern is anchored to @cms.hhs.gov to
-- avoid an accidental match on an unrelated contractor address; the fullname
-- pattern requires a space between "elizabeth" and "schweinsberg" so phrases
-- like "Elizabeth Smith and Carol Schweinsberg" do not match.
--
-- A RAISE NOTICE at the end logs how many rows matched each path so the swap
-- result is easy to verify post-deploy. If no row matches in a given
-- environment (e.g. fresh local DBs without her record), Elizabeth is
-- provisioned later through the admin panel.
--
-- Order matters: Elizabeth's HHS_ADMIN assignment runs first so the bulk
-- ADMIN -> OWNER update below skips her by virtue of her role no longer being
-- ADMIN.

DO $$
DECLARE
    n_email          int := 0;
    n_name_fallback  int := 0;
    n_owner          int := 0;
    n_readonly       int := 0;
    n_hhs_admin_post int := 0;
BEGIN
    UPDATE public.users
       SET role = 'HHS_ADMIN'
     WHERE role = 'ADMIN'
       AND LOWER(email) LIKE 'elizabeth.schweinsberg%@cms.hhs.gov';
    GET DIAGNOSTICS n_email = ROW_COUNT;

    IF n_email = 0 THEN
        UPDATE public.users
           SET role = 'HHS_ADMIN'
         WHERE role = 'ADMIN'
           AND (
                LOWER(fullname) LIKE 'elizabeth % schweinsberg'
             OR LOWER(fullname) LIKE 'elizabeth schweinsberg'
             OR LOWER(fullname) LIKE 'schweinsberg, elizabeth%'
               );
        GET DIAGNOSTICS n_name_fallback = ROW_COUNT;
    END IF;

    UPDATE public.users
       SET role = 'OWNER'
     WHERE role = 'ADMIN';
    GET DIAGNOSTICS n_owner = ROW_COUNT;

    UPDATE public.users
       SET role = 'HHS_READONLY_ADMIN'
     WHERE role = 'READONLY_ADMIN';
    GET DIAGNOSTICS n_readonly = ROW_COUNT;

    SELECT count(*) INTO n_hhs_admin_post
      FROM public.users
     WHERE role = 'HHS_ADMIN';

    RAISE NOTICE 'role swap complete: Elizabeth match by email=%, by name fallback=%, ADMIN -> OWNER=%, READONLY_ADMIN -> HHS_READONLY_ADMIN=%, total HHS_ADMIN rows post-swap=%',
        n_email, n_name_fallback, n_owner, n_readonly, n_hhs_admin_post;
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
