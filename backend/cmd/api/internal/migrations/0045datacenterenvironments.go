package migrations

func init() {
	getMigrator().AppendMigration(
		"create datacenterenvironments mapping table, seed it, rename OPDC, add MAG and data-center-gov function sets",
		`
-- ZTMF scores a system by matching its datacenterenvironment against the
-- functions catalog (functions.datacenterenvironment). That catalog is keyed to
-- a small, fixed vocabulary. Real inventory values (HHS ScoreCard exports and
-- other OpDivs) use free-text environment names that are not in it, so the
-- scoring join returned no functions and those systems scored 0.00 (ztmf#392).
--
-- This reference table decouples the value stored on a system (kept untouched so
-- each OpDiv keeps its own reporting label) from the scoring vocabulary:
--
--   datacenterenvironment  the raw value as stored on fismasystems.datacenterenvironment
--   category               the reporting / dropdown bucket the raw value belongs to
--   scoring_key            the functions.datacenterenvironment set to score against
--                          (NULL = not scored, e.g. the legacy DECOMMISSIONED marker)
--   selectable             TRUE for values offered in the new/edit-system dropdown
--   ordr                   dropdown ordering
--
-- Every scored environment is a first-class function set with its own key, so
-- scoring_key is an identity mapping today. It stays a distinct column so a
-- future divergence (e.g. a shared cloud base) can point several environments at
-- one set by changing data, not code. All agency-specific vocabulary lives in
-- these rows, never in code, so a new OpDiv or deployment adds rows here.

CREATE TABLE IF NOT EXISTS public.datacenterenvironments (
    datacenterenvironment VARCHAR(255) PRIMARY KEY,
    category              VARCHAR(255) NOT NULL,
    scoring_key           VARCHAR(255),
    selectable            BOOLEAN NOT NULL DEFAULT FALSE,
    ordr                  SMALLINT NOT NULL DEFAULT 0
);

COMMENT ON TABLE public.datacenterenvironments IS 'Maps a system''s raw datacenterenvironment to a reporting category and the functions.datacenterenvironment set used to score it (ztmf#392). Reference data - extend per deployment.';

-- Rename the cryptic OPDC ("Other People''s Data Center") function set to the
-- self-describing data-center-contractor. Pure key rename: functionids (and every
-- score that references them) are unchanged. On a fresh local/test database the
-- functions catalog is empty at migration time and this - like the two copies
-- below - affects zero rows; the empire seed supplies its own vocabulary.
UPDATE public.functions SET datacenterenvironment = 'data-center-contractor'
 WHERE datacenterenvironment = 'OPDC';

-- MAG is a first-class copy of the CMS Azure/MAG question set. It gets its own
-- functionids so MAG (non-CMS) systems score against it independently; stripping
-- the CMS-specific wording is a follow-up content edit on these rows. The copy
-- correlates options by function name, which is unique within an environment.
INSERT INTO public.functions (function, description, datacenterenvironment, ordr, questionid, pillarid)
SELECT function, description, 'MAG', ordr, questionid, pillarid
  FROM public.functions WHERE datacenterenvironment = 'CMS-Cloud-MAG';

INSERT INTO public.functionoptions (functionid, score, optionname, description)
SELECT newf.functionid, fo.score, fo.optionname, fo.description
  FROM public.functions newf
  JOIN public.functions oldf
    ON oldf.datacenterenvironment = 'CMS-Cloud-MAG'
   AND newf.datacenterenvironment = 'MAG'
   AND newf.function = oldf.function
  JOIN public.functionoptions fo ON fo.functionid = oldf.functionid;

-- data-center-gov starts as a copy of the data-center-contractor (former OPDC)
-- question set - the HHS ScoreCard base Elizabeth identified. Distinct functionids
-- let gov and contractor diverge later by editing these rows.
INSERT INTO public.functions (function, description, datacenterenvironment, ordr, questionid, pillarid)
SELECT function, description, 'data-center-gov', ordr, questionid, pillarid
  FROM public.functions WHERE datacenterenvironment = 'data-center-contractor';

INSERT INTO public.functionoptions (functionid, score, optionname, description)
SELECT newf.functionid, fo.score, fo.optionname, fo.description
  FROM public.functions newf
  JOIN public.functions oldf
    ON oldf.datacenterenvironment = 'data-center-contractor'
   AND newf.datacenterenvironment = 'data-center-gov'
   AND newf.function = oldf.function
  JOIN public.functionoptions fo ON fo.functionid = oldf.functionid;

-- Canonical categories: the values offered in the system dropdown going forward.
-- Stored value, reporting category, and scoring key are the same (identity).
INSERT INTO public.datacenterenvironments (datacenterenvironment, category, scoring_key, selectable, ordr) VALUES
    ('CMS-Cloud-AWS',          'CMS-Cloud-AWS',          'CMS-Cloud-AWS',          TRUE, 10),
    ('CMS-Cloud-MAG',          'CMS-Cloud-MAG',          'CMS-Cloud-MAG',          TRUE, 20),
    ('CMSDC',                  'CMSDC',                  'CMSDC',                  TRUE, 30),
    ('AWS',                    'AWS',                    'AWS',                    TRUE, 40),
    ('MAG',                    'MAG',                    'MAG',                    TRUE, 50),
    ('SaaS',                   'SaaS',                   'SaaS',                   TRUE, 60),
    ('Other',                  'Other',                  'Other',                  TRUE, 70),
    ('data-center-gov',        'data-center-gov',        'data-center-gov',        TRUE, 80),
    ('data-center-contractor', 'data-center-contractor', 'data-center-contractor', TRUE, 90)
ON CONFLICT DO NOTHING;

-- Known raw-value aliases: free-text inventory strings already present on systems
-- (HHS ScoreCard exports, legacy CMS values). Not selectable - they resolve an
-- existing system to a category/scoring key but are not offered for new systems.
-- DECOMMISSIONED is a legacy marker, not an environment: scoring_key NULL leaves
-- those systems out of scoring (pending cleanup, ztmf#392 follow-up).
INSERT INTO public.datacenterenvironments (datacenterenvironment, category, scoring_key, selectable, ordr) VALUES
    ('OPDC',                                         'data-center-contractor', 'data-center-contractor', FALSE, 0),
    ('Data Center: Gov-Owned, Multi-tenant',         'data-center-gov',        'data-center-gov',        FALSE, 0),
    ('Data Center: Contractor-Owned, Single-tenant', 'data-center-contractor', 'data-center-contractor', FALSE, 0),
    ('AWS (incl GovCloud)',                          'AWS',                    'AWS',                    FALSE, 0),
    ('Azure (Commercial or MAG)',                    'MAG',                    'MAG',                    FALSE, 0),
    ('Cloud-CMS-AWS',                                'CMS-Cloud-AWS',          'CMS-Cloud-AWS',          FALSE, 0),
    ('Cloud-CMS-Azure',                              'CMS-Cloud-MAG',          'CMS-Cloud-MAG',          FALSE, 0),
    ('DECOMMISSIONED',                               'DECOMMISSIONED',         NULL,                     FALSE, 0)
ON CONFLICT DO NOTHING;
		`,
		`
-- Drop the copied MAG and data-center-gov sets. Their functionoptions must go
-- first (scores FK to functionoptions). A rollback after live scores have been
-- re-pointed onto these sets requires re-pointing them back first; this down step
-- assumes rollback happens before that data surgery.
DELETE FROM public.functionoptions
 WHERE functionid IN (SELECT functionid FROM public.functions
                       WHERE datacenterenvironment IN ('MAG', 'data-center-gov'));
DELETE FROM public.functions
 WHERE datacenterenvironment IN ('MAG', 'data-center-gov');

-- Reverse the rename. Assumes data-center-contractor still denotes the single
-- renamed OPDC set (the state this migration leaves it in).
UPDATE public.functions SET datacenterenvironment = 'OPDC'
 WHERE datacenterenvironment = 'data-center-contractor';

DROP TABLE IF EXISTS public.datacenterenvironments;
		`)
}
