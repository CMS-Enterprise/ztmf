package migrations

func init() {
	getMigrator().AppendMigration(
		"create systemattributes reference table and seed canonical HHS metadata vocabulary",
		`
-- Canonical allowed values per FISMA system attribute (ztmf#395). Serves the
-- backend-driven select vocabulary for the extended HHS metadata fields (the
-- reference-data pattern #392 established for datacenterenvironments) and backs
-- the friendly server-side validation of enum writes.
--
-- The columns themselves are already typed + constrained by ztmf#433 (booleans,
-- CHECK-enums, cloud_service_model text[]); that migration is the authoritative
-- validator at the DB. This table's canonical (selectable=TRUE) set MUST stay in
-- sync with those CHECK constraints - it is the single vocabulary source the
-- endpoint serves and the app validates against, and the #433 CHECK is the DB
-- backstop. So we seed ONLY canonical values here: seeding an off-canon "legacy"
-- value would make app validation accept a write the #433 CHECK then rejects.
--
--   field        the fismasystems column the value belongs to
--   value        an atomic allowed value; '' is reserved for a field-level help
--                row that carries only a description (never a real value)
--   description  optional per-value or per-field help text (e.g. the legacy
--                definition), surfaced by the frontend as hover help
--   selectable   TRUE for values offered in the add/edit dropdown; FALSE marks
--                the non-value help rows
--   ordr         dropdown ordering within a field

CREATE TABLE IF NOT EXISTS public.systemattributes (
    field       VARCHAR(64)  NOT NULL,
    value       VARCHAR(255) NOT NULL,
    description VARCHAR(1024),
    selectable  BOOLEAN NOT NULL DEFAULT TRUE,
    ordr        SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (field, value)
);

COMMENT ON TABLE public.systemattributes IS 'Canonical allowed values per FISMA system attribute; backend-driven select vocab + friendly write validation (ztmf#395). Kept in sync with the ztmf#433 CHECK constraints. Reference data - extend per deployment.';

-- Canonical, selectable values (the set offered in the frontend selects and the
-- set app validation accepts). Order matches the display order confirmed on
-- ztmf#395. cloud_service_model is the four ATOMIC parts; a system's stored
-- text[] is validated element-by-element against them (ztmf#433 stores the
-- decomposed array, not a slash-joined string). hva/cloud_system/legacy rows
-- are the Yes/No labels the select renders - the columns are booleans, so these
-- are not used to validate a write, only to build the dropdown.
INSERT INTO public.systemattributes (field, value, selectable, ordr) VALUES
    ('fips',                'Low',                     TRUE, 10),
    ('fips',                'Moderate',                TRUE, 20),
    ('fips',                'High',                    TRUE, 30),

    ('system_type',         'Major Application',       TRUE, 10),
    ('system_type',         'Minor Application',       TRUE, 20),
    ('system_type',         'Minor Standalone',        TRUE, 30),
    ('system_type',         'General Support System',  TRUE, 40),
    ('system_type',         'Enterprise',              TRUE, 50),
    ('system_type',         'Local',                   TRUE, 60),
    ('system_type',         'Other',                   TRUE, 70),

    ('hva',                 'Yes',                     TRUE, 10),
    ('hva',                 'No',                      TRUE, 20),

    ('cloud_system',        'Yes',                     TRUE, 10),
    ('cloud_system',        'No',                      TRUE, 20),

    ('cloud_service_model', 'IaaS',                    TRUE, 10),
    ('cloud_service_model', 'PaaS',                    TRUE, 20),
    ('cloud_service_model', 'SaaS',                    TRUE, 30),
    ('cloud_service_model', 'Other',                   TRUE, 40),

    ('goco_coco_gogo',      'GOCO',                    TRUE, 10),
    ('goco_coco_gogo',      'COCO',                    TRUE, 20),
    ('goco_coco_gogo',      'GOGO',                    TRUE, 30),

    ('system_operator',     'Agency',                  TRUE, 10),
    ('system_operator',     'Contractor',              TRUE, 20),

    ('legacy',              'Yes',                     TRUE, 10),
    ('legacy',              'No',                      TRUE, 20)
ON CONFLICT DO NOTHING;

-- Field-level help row: value '' carries the field's definition for hover help.
-- TODO(ztmf#395): confirm the final legacy definition wording with danielbowne
-- before this ships. Straw-man from the #395 thread below.
INSERT INTO public.systemattributes (field, value, description, selectable, ordr) VALUES
    ('legacy', '', 'A legacy system runs end-of-life/unsupported software or OS, depends on an unsupported vendor, or has no active development team.', FALSE, 0)
ON CONFLICT DO NOTHING;
		`,
		`
DROP TABLE IF EXISTS public.systemattributes;
		`)
}
