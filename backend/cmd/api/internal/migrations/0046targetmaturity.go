package migrations

func init() {
	getMigrator().AppendMigration(
		"add target maturity tier and justification to fismasystems",
		`
-- Risk-based target maturity level per system (#398, GAO audit response).
-- Stored as the tier NAME, not a CISA stage number: the app's internal score
-- scale is 1-5 and Tier() maps 3.10-4.09 to Advanced, so a raw integer target
-- (e.g. 3) must never be compared against systemscore directly. Storing the
-- name makes tier-vs-tier the only expressible comparison.
-- Nullable on purpose: NULL means "no ISSO has asserted a target yet" and the
-- UI presents the Advanced default. No backfill - fabricating an explicit
-- assertion for every system would defeat the audit purpose.

ALTER TABLE public.fismasystems
  ADD COLUMN IF NOT EXISTS target_maturity_tier varchar(20)
    CONSTRAINT fismasystems_target_maturity_tier_check
    CHECK (target_maturity_tier IN ('Initial', 'Advanced', 'Optimal')),
  ADD COLUMN IF NOT EXISTS target_maturity_justification varchar(1000);
		`,
		`
ALTER TABLE public.fismasystems
  DROP COLUMN IF EXISTS target_maturity_tier,
  DROP COLUMN IF EXISTS target_maturity_justification;
		`)
}
