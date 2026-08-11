package migrations

func init() {
	appendMigration(
		"create reducedpillarscopes and seed the SaaS rule",
		`
-- The reduced-pillar rule as data (ztmf#545, ztmf-misc#289). A row means:
-- systems scored under scoring_key exclude pillarid from every data call whose
-- deadline is on or after the anchor call's deadline. The predicate that reads
-- this table (reducedPillarScopeSQL in internal/model/pillarscope.go) is shared
-- by the progress counts, the export, the scoring aggregate and the
-- questionnaire, so a future reduced-scope environment is a row here, not a
-- code change.
--
-- The anchor is a datacalls FK rather than a stored date so the boundary is an
-- auditable reference to the cycle where the rule took effect, and so a
-- mis-dated or FY26-named call created later cannot move it - the failure mode
-- the interim name-prefix predicate this replaces had to defend against.
-- ON DELETE CASCADE because a rule without its anchor is undefined; deleting a
-- data call is already an admin-only destructive act and takes its rules along.
CREATE TABLE reducedpillarscopes (
    scoring_key          TEXT NOT NULL,
    pillarid             INT  NOT NULL REFERENCES pillars(pillarid),
    effective_datacallid INT  NOT NULL REFERENCES datacalls(datacallid) ON DELETE CASCADE,
    PRIMARY KEY (scoring_key, pillarid)
);

-- Seed the SaaS rule: Devices and Applications leave scope from the FY26 cycle
-- onward. The name-prefix lookup below is the LAST use of name matching: it
-- runs once, here, to locate the anchor in the data each environment already
-- has, mirroring the prefixes the interim predicate matched (both year forms,
-- trimmed and uppercased). Earliest by (deadline, datacallid) is "the first
-- FY26 cycle", the same choice the FY26 rollover hardcode pins.
--
-- An environment with no FY26-named call at migration time seeds nothing: the
-- ephemeral local/test databases get their rule rows from the populate seed
-- (_test_data_empire.sql), which owns the fixture cycles and runs after
-- migrations; a deployed environment without the call gets the rule when the
-- row is inserted alongside the cycle that needs it.
INSERT INTO reducedpillarscopes (scoring_key, pillarid, effective_datacallid)
SELECT 'SaaS', p.pillarid, anchor.datacallid
FROM pillars p
CROSS JOIN (
    SELECT datacallid FROM datacalls
     WHERE UPPER(TRIM(datacall)) LIKE 'FY2026%' OR UPPER(TRIM(datacall)) LIKE 'FY26%'
     ORDER BY deadline ASC, datacallid ASC
     LIMIT 1
) anchor
WHERE p.pillar IN ('Devices', 'Applications');
`,
		`
DROP TABLE reducedpillarscopes;
`,
	)
}
