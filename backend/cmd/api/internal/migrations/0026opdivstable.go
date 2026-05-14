package migrations

func init() {
	getMigrator().AppendMigration(
		"create opdivs reference table and seed HHS + OpDivs",
		`
CREATE TABLE IF NOT EXISTS public.opdivs (
    opdiv_id   SERIAL PRIMARY KEY,
    code       VARCHAR(16) NOT NULL,
    name       VARCHAR(128) NOT NULL,
    is_parent  BOOLEAN NOT NULL DEFAULT FALSE,
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS opdivs_code_lower_idx
    ON public.opdivs (LOWER(code))
    WHERE active = TRUE;

COMMENT ON TABLE public.opdivs IS 'HHS Operating Divisions (parent department + sister OpDivs). Reference data for multi-tenant scoping of ZTMF.';
COMMENT ON COLUMN public.opdivs.code IS 'Short code used in URLs, dropdowns, and external references (e.g. HHS, CMS, CDC).';
COMMENT ON COLUMN public.opdivs.is_parent IS 'TRUE for the HHS parent row, FALSE for sister OpDivs. Lets HHS_ADMIN derive "all OpDivs" without a magic sentinel.';

INSERT INTO public.opdivs (code, name, is_parent, active) VALUES
    ('HHS',    'Department of Health and Human Services',                TRUE,  TRUE),
    ('CMS',    'Centers for Medicare & Medicaid Services',               FALSE, TRUE),
    ('CDC',    'Centers for Disease Control and Prevention',             FALSE, TRUE),
    ('NIH',    'National Institutes of Health',                          FALSE, TRUE),
    ('FDA',    'Food and Drug Administration',                           FALSE, TRUE),
    ('HRSA',   'Health Resources and Services Administration',           FALSE, TRUE),
    ('IHS',    'Indian Health Service',                                  FALSE, TRUE),
    ('SAMHSA', 'Substance Abuse and Mental Health Services Admin',       FALSE, TRUE),
    ('ACF',    'Administration for Children and Families',               FALSE, TRUE),
    ('ACL',    'Administration for Community Living',                    FALSE, TRUE),
    ('AHRQ',   'Agency for Healthcare Research and Quality',             FALSE, TRUE),
    ('ATSDR',  'Agency for Toxic Substances and Disease Registry',       FALSE, TRUE)
ON CONFLICT DO NOTHING;
        `,
		`
DROP INDEX IF EXISTS public.opdivs_code_lower_idx;
DROP TABLE IF EXISTS public.opdivs;
        `)
}
