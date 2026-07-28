package migrations

func init() {
	appendMigration(
		"retype fismasystems HHS metadata columns: booleans, enum CHECKs, and a cloud_service_model text[]",
		`
-- ztmf#433: the HHS onboarding metadata columns were added as free-text
-- varchar(255) in 0043. This migration makes them typed and self-validating at
-- the database layer:
--   - booleans (hva, cloud_system, legacy): varchar -> boolean, null = unknown
--     (the epic rule is do NOT coerce a missing value to No);
--   - enums (fips, system_type, system_operator, goco_coco_gogo): stay varchar
--     with a CHECK constraint (house style, matching 0046/0048 - avoids the
--     native ENUM ALTER TYPE dance);
--   - cloud_service_model: varchar combo -> text[], with a CHECK on the elements.
--
-- Existing rows are canonicalized in place first so the type casts and CHECKs
-- cannot fail the migration. Postgres data is mostly NULL for these fields today
-- (HHS not yet loaded; CMS rows NULL), so this touches few rows, but the mapping
-- covers every off-canon spelling catalogued in ztmf#395. Anything still
-- unmappable is set to NULL ("not yet captured") rather than blocking startup.
--
-- The vocabulary endpoint, friendly 400 messages, and the cloud_system=No
-- cross-field rule are ztmf#395, layered on top of these typed columns.
--
-- NOTE: this changes the JSON API contract (bool / array instead of strings) and
-- breaks the PG->Snowflake SDL export (CORE.ZTMF_FISMASYSTEMS) in the private
-- ztmf-insights repo. Both are expected and coordinated (danielbowne accepted a
-- temporary SDL break; the export is patched downstream).

-- 1. Canonicalize enum values in place, collapsing the ztmf#395 variants and
--    nulling anything off-canon so the CHECKs below hold.
-- Fold on lower(btrim(...)) like the other three enums below, so casing and
-- stray whitespace ('major application', ' Major Application') canonicalize
-- instead of silently dropping to NULL.
UPDATE public.fismasystems SET system_type = CASE lower(btrim(system_type))
    WHEN 'minor standalone'               THEN 'Minor Standalone'
    WHEN 'minor stand-alone'              THEN 'Minor Standalone'
    WHEN 'minor application (standalone)' THEN 'Minor Standalone'
    WHEN 'minor application (standlone)'  THEN 'Minor Standalone'
    WHEN 'minor application'              THEN 'Minor Application'
    WHEN 'minor application (child)'      THEN 'Minor Application'
    WHEN 'major application'              THEN 'Major Application'
    WHEN 'general support system'         THEN 'General Support System'
    WHEN 'general support services'       THEN 'General Support System'
    WHEN 'enterprise'                     THEN 'Enterprise'
    WHEN 'local'                          THEN 'Local'
    WHEN 'other'                          THEN 'Other'
    WHEN 'others -'                       THEN 'Other'
    ELSE NULL
END
WHERE system_type IS NOT NULL;

UPDATE public.fismasystems SET fips = CASE
    WHEN initcap(btrim(fips)) IN ('Low', 'Moderate', 'High') THEN initcap(btrim(fips))
    ELSE NULL
END
WHERE fips IS NOT NULL;

UPDATE public.fismasystems SET system_operator = CASE
    WHEN initcap(btrim(system_operator)) IN ('Agency', 'Contractor') THEN initcap(btrim(system_operator))
    ELSE NULL
END
WHERE system_operator IS NOT NULL;

UPDATE public.fismasystems SET goco_coco_gogo = CASE
    WHEN upper(btrim(goco_coco_gogo)) = 'GOVO'                     THEN 'GOGO'
    WHEN upper(btrim(goco_coco_gogo)) IN ('GOCO', 'COCO', 'GOGO')  THEN upper(btrim(goco_coco_gogo))
    ELSE NULL
END
WHERE goco_coco_gogo IS NOT NULL;

-- 2. Normalize cloud_service_model to a sorted, slash-joined canonical combo
--    (split on / , ;, case-fold each token, drop unknowns), so the text[] cast
--    below yields clean canonical elements. Empty result -> NULL.
--    Deliberately two-step (normalize-to-string here, then string_to_array in
--    step 4): the column is still varchar at this point and cannot hold an
--    array, so the '/' round-trip is required, not accidental. Both steps share
--    the '/' delimiter on purpose - do not "simplify" one without the other.
UPDATE public.fismasystems SET cloud_service_model = (
    SELECT string_agg(x, '/' ORDER BY x)
    FROM (
        SELECT DISTINCT CASE lower(btrim(t))
            WHEN 'iaas'  THEN 'IaaS'
            WHEN 'paas'  THEN 'PaaS'
            WHEN 'saas'  THEN 'SaaS'
            WHEN 'other' THEN 'Other'
            ELSE NULL
        END AS x
        FROM regexp_split_to_table(cloud_service_model, '[/,;]+') AS t
    ) mapped
    WHERE x IS NOT NULL
)
WHERE cloud_service_model IS NOT NULL AND btrim(cloud_service_model) <> '';

-- 3. Retype the booleans. lower()/btrim() folds the YES/yes/NO/no variants;
--    anything else becomes NULL (unknown), never false.
ALTER TABLE public.fismasystems
    ALTER COLUMN hva          TYPE boolean USING (CASE WHEN lower(btrim(hva))          = 'yes' THEN true WHEN lower(btrim(hva))          = 'no' THEN false ELSE NULL END),
    ALTER COLUMN cloud_system TYPE boolean USING (CASE WHEN lower(btrim(cloud_system)) = 'yes' THEN true WHEN lower(btrim(cloud_system)) = 'no' THEN false ELSE NULL END),
    ALTER COLUMN legacy       TYPE boolean USING (CASE WHEN lower(btrim(legacy))       = 'yes' THEN true WHEN lower(btrim(legacy))       = 'no' THEN false ELSE NULL END);

-- 4. Retype cloud_service_model to text[].
ALTER TABLE public.fismasystems
    ALTER COLUMN cloud_service_model TYPE text[] USING (
        CASE WHEN cloud_service_model IS NULL OR btrim(cloud_service_model) = '' THEN NULL
             ELSE string_to_array(cloud_service_model, '/')
        END
    );

-- 5. Constrain the enum columns and the array elements to the canonical sets
--    (NULL-tolerant: null = not yet captured stays valid). These lists must stay
--    in sync with the ztmf#395 systemattributes selectable=TRUE seed.
ALTER TABLE public.fismasystems
    DROP CONSTRAINT IF EXISTS fismasystems_fips_check,
    DROP CONSTRAINT IF EXISTS fismasystems_system_type_check,
    DROP CONSTRAINT IF EXISTS fismasystems_system_operator_check,
    DROP CONSTRAINT IF EXISTS fismasystems_goco_coco_gogo_check,
    DROP CONSTRAINT IF EXISTS fismasystems_cloud_service_model_check;

ALTER TABLE public.fismasystems
    ADD CONSTRAINT fismasystems_fips_check
        CHECK (fips IS NULL OR fips IN ('Low', 'Moderate', 'High')),
    ADD CONSTRAINT fismasystems_system_type_check
        CHECK (system_type IS NULL OR system_type IN ('Major Application', 'Minor Application', 'Minor Standalone', 'General Support System', 'Enterprise', 'Local', 'Other')),
    ADD CONSTRAINT fismasystems_system_operator_check
        CHECK (system_operator IS NULL OR system_operator IN ('Agency', 'Contractor')),
    ADD CONSTRAINT fismasystems_goco_coco_gogo_check
        CHECK (goco_coco_gogo IS NULL OR goco_coco_gogo IN ('GOCO', 'COCO', 'GOGO')),
    ADD CONSTRAINT fismasystems_cloud_service_model_check
        CHECK (cloud_service_model IS NULL OR cloud_service_model <@ ARRAY['IaaS', 'PaaS', 'SaaS', 'Other']::text[]);
		`,
		`
-- Reverse the retype. Drop the CHECKs, then cast the typed columns back to
-- varchar: booleans render as 'Yes'/'No' (NULL stays NULL), the array re-joins
-- with '/'. The original raw off-canon spellings are not recoverable - the up
-- migration canonicalized them in place - so this restores the canonical form,
-- not the pre-0052 free text.
ALTER TABLE public.fismasystems
    DROP CONSTRAINT IF EXISTS fismasystems_fips_check,
    DROP CONSTRAINT IF EXISTS fismasystems_system_type_check,
    DROP CONSTRAINT IF EXISTS fismasystems_system_operator_check,
    DROP CONSTRAINT IF EXISTS fismasystems_goco_coco_gogo_check,
    DROP CONSTRAINT IF EXISTS fismasystems_cloud_service_model_check;

ALTER TABLE public.fismasystems
    ALTER COLUMN hva          TYPE varchar(255) USING (CASE WHEN hva          IS TRUE THEN 'Yes' WHEN hva          IS FALSE THEN 'No' ELSE NULL END),
    ALTER COLUMN cloud_system TYPE varchar(255) USING (CASE WHEN cloud_system IS TRUE THEN 'Yes' WHEN cloud_system IS FALSE THEN 'No' ELSE NULL END),
    ALTER COLUMN legacy       TYPE varchar(255) USING (CASE WHEN legacy       IS TRUE THEN 'Yes' WHEN legacy       IS FALSE THEN 'No' ELSE NULL END),
    ALTER COLUMN cloud_service_model TYPE varchar(255) USING (array_to_string(cloud_service_model, '/'));
		`)
}
