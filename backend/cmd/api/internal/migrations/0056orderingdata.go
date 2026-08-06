package migrations

func init() {
	appendMigration(
		"populate pillars.ordr and questions.ordr with the canonical questionnaire order",
		`
-- pillars.ordr and questions.ordr have existed since 0002 but were never
-- populated: every row is 0 except questions 41-47, the legacy "Future Plan..."
-- questions that carry 901-907 so they sink to the bottom. With every other
-- row tied at 0 the API's ORDER BY collapsed to whatever the planner felt like
-- returning, so the questionnaire's real order has been carried by the frontend
-- instead - PILLAR_ORDER and PILLAR_FUNCTION_MAP in ztmf-ui's src/constants.ts,
-- which re-sorts the API payload client-side before rendering.
--
-- That works only for the one consumer that owns the map. The spreadsheet
-- export, the score diff, and any future API client all read the same unordered
-- payload and have no way to recover the intended sequence. Worse, the ordering
-- they do get is heap order, which is stable only until something rewrites the
-- rows - exactly the failure mode that scrambled the answer choices in
-- ztmf-misc#279, whose related-finding thread is what surfaced this.
--
-- So: move the order into the data, where every consumer sees it.
--
-- Why the frontend map is the source. It is not a guess that needs review - it
-- is the order the questionnaire has rendered in production for years, matching
-- the CISA Zero Trust Maturity Model's pillar sequence (Identity, Devices,
-- Networks, Applications, Data, then the Cross-Cutting capabilities) and, within
-- each pillar, the model's function sequence ending in the three cross-cutting
-- capabilities (Visibility & Analytics, Automation & Orchestration, Governance).
-- Transcribing it is a lift-and-shift of an already-approved order, not a new
-- editorial decision.
--
-- Why the data lives in a migration rather than a seed script or a one-off SQL
-- run: it ships in the same versioned, reviewed, single-PR artifact as the
-- ORDER BY changes that depend on it, it runs identically in local, test, dev
-- and prod, and it is reversible.
--
-- Scheme: questions.ordr = pillar_rank * 100 + function_index, both 1-based
-- (Identity 101-107, Devices 201-207, Networks 301-307, Applications 401-408,
-- Data 501-508, CrossCutting 601-603). The pillar prefix is redundant with the
-- ORDER BY (which sorts pillars.ordr first) but makes a raw questions row
-- self-describing, and leaves gaps so a function can be inserted mid-pillar
-- without renumbering. All values stay well under the legacy 900 band and
-- inside smallint.
--
-- Questions are keyed to the map by their function's NAME, not by id: each
-- questionnaire edition (cloud, on-prem, ...) has its own functions row, but all
-- editions of a given question share the questionid and the function name, so
-- the name is the stable business key. Verified against a prod copy: all 40
-- non-legacy questions resolve to exactly one distinct function name and all 40
-- names are present in the map, no leftovers on either side.
--
-- Unmatched questions keep ordr = 0 by design. The empire seed used by the local
-- and test databases has fictional function names that will never match the map,
-- and the WHERE ordr = 0 guard likewise leaves the 901-907 legacy values alone.
-- Neither case is a problem: the queries this migration supports all carry a
-- questions.questionid tiebreaker, so rows sharing ordr = 0 still come back in a
-- stable order, just not a curated one.
UPDATE pillars p
SET ordr = v.ordr
FROM (VALUES
    ('Identity',     1),
    ('Devices',      2),
    ('Networks',     3),
    ('Applications', 4),
    ('Data',         5),
    ('CrossCutting', 6)
) AS v(pillar, ordr)
WHERE p.pillar = v.pillar;

-- Deterministic by construction rather than by hope: a question is ranked by
-- the MINIMUM matching rank among its functions' names. On prod data every
-- question matches exactly one name (verified: 40/40 single-name), so min() is
-- the identity; if a future edition ever forked a question's function name,
-- min() still yields one stable, re-runnable answer instead of letting the
-- planner pick a VALUES row arbitrarily.
UPDATE questions q
SET ordr = ranked.ordr
FROM (
  SELECT f.questionid, min(r.ordr) AS ordr
  FROM functions f
  JOIN (VALUES
    ('Authentication-Users',                101),
    ('IdentityStores-Users',                102),
    ('RiskAssessment',                      103),
    ('AccessManagement',                    104),
    ('Identity-VisibilityAnalytics',        105),
    ('Identity-AutomationOrchestration',    106),
    ('Identity-Governance',                 107),
    ('PolicyEnforcement',                   201),
    ('AssetRiskManagement',                 202),
    ('ResourceAccess',                      203),
    ('Device-ThreatProtection',             204),
    ('Device-VisibilityAnalytics',          205),
    ('Device-AutomationOrchestration',      206),
    ('Device-Governance',                   207),
    ('NetworkSegmentation',                 301),
    ('NetworkTrafficManagement',            302),
    ('Network-Encryption',                  303),
    ('NetworkResilience',                   304),
    ('Network-VisibilityAnalytics',         305),
    ('Network-AutomationOrchestration',     306),
    ('Network-Governance',                  307),
    ('AccessAuthorization-Users',           401),
    ('Application-ThreatProtection',        402),
    ('AccessibleApplications',              403),
    ('SecureDevDeployWorkflow',             404),
    ('Application-SecurityTesting',         405),
    ('Application-VisibilityAnalytics',     406),
    ('Application-AutomationOrchestration', 407),
    ('Application-Governance',              408),
    ('DataInventoryManagement',             501),
    ('DataCategorization',                  502),
    ('DataAvailability',                    503),
    ('DataAccess',                          504),
    ('DataEncryption',                      505),
    ('Data-VisibilityAnalytics',            506),
    ('Data-AutomationOrchestration',        507),
    ('Data-Governance',                     508),
    ('Cross-VisibilityAnalytics',           601),
    ('Cross-AutomationOrchestration',       602),
    ('Cross-Governance',                    603)
  ) AS r(function_name, ordr) ON r.function_name = f.function
  WHERE f.questionid IS NOT NULL
  GROUP BY f.questionid
) AS ranked
WHERE q.questionid = ranked.questionid
  AND q.ordr = 0;
`,
		`
-- Reverses the population only. The legacy 901-907 band predates this migration
-- and is not ours to clear, so the down guard mirrors the up guard from the
-- other side: everything this migration could have written is below 900.
UPDATE pillars SET ordr = 0;
UPDATE questions SET ordr = 0 WHERE ordr < 900;
`,
	)
}
