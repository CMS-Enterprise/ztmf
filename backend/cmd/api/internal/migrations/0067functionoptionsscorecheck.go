package migrations

func init() {
	appendMigration(
		"constrain functionoptions.score to the 1-4 answer scale",
		`
-- ztmf-misc#434. Answers are 1-4 (Traditional..Optimal) and the HHS formula
-- (ztmf-misc#433) scores a pillar as their average times 1.25, so an option
-- outside 1..4 would put a pillar off the 1.25-5.00 scale. FunctionOption.validate
-- already rejects such a write with a 400; this CHECK is the DB backstop for any
-- path that skips it (a load script, a hand-run UPDATE).
--
-- Safe on populated tables: score is NOT NULL and every existing row is 1-4
-- (checked on a prod-shaped copy before writing this). Guarded by a DO block
-- because Postgres has no ADD CONSTRAINT IF NOT EXISTS, so a retry is a no-op.
SET LOCAL lock_timeout = '10s';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'functionoptions_score_check'
          AND conrelid = 'public.functionoptions'::regclass
    ) THEN
        ALTER TABLE public.functionoptions
            ADD CONSTRAINT functionoptions_score_check
            CHECK (score BETWEEN 1 AND 4);
    END IF;
END $$;
`,
		`ALTER TABLE public.functionoptions DROP CONSTRAINT IF EXISTS functionoptions_score_check;`,
	)
}
