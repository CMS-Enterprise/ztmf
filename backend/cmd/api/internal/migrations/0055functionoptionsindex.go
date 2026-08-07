package migrations

func init() {
	appendMigration(
		"add functionoptions(functionid) index for the data-call export",
		`
-- The data-call export (#528 re-anchor of FindAnswers) left-joins scores with
-- a per-row subquery over functionoptions keyed on functionid. Postgres cannot
-- fold that IN-subquery into the hash join, so it runs as a per-candidate-row
-- Join Filter - and functionoptions has no functionid index (only its primary
-- key), making each evaluation a sequential scan.
--
-- Measured on a prod-scale copy (ztmf#531, 1,530 systems / 192k scores): the
-- whole-call FY26 export performs that scan 1,556,120 times, 47.7s of database
-- time before the spreadsheet is even rendered - retry/abandon territory, and
-- near typical gateway idle timeouts. With this index the same export runs in
-- 2.0s (23x).
--
-- functionoptions is a small, static, read-mostly catalogue (1,728 rows; it
-- changes only when a questionnaire edition is edited), so the write cost is
-- negligible and no partial or composite form is warranted.
CREATE INDEX IF NOT EXISTS functionoptions_functionid_idx
    ON public.functionoptions (functionid);
`,
		`DROP INDEX IF EXISTS functionoptions_functionid_idx;`,
	)
}
