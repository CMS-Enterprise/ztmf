package migrations

func init() {
	appendMigration(
		"add scores.last_updated_at for score-progress last-updated",
		`
-- ztmf-misc#426. /scores/progress derived last-updated from a per-score-row lateral
-- onto events, 98% of that query's cost. Persist it on the row instead, written
-- by Save/Confirm in the same statement that sets status = 'done'.
--
-- Nullable with no default: carried-forward (copyPreviousScores) and imported
-- rows must stay NULL. The backfill is a separate migration so this one holds
-- its ACCESS EXCLUSIVE lock only for the catalog change.
SET LOCAL lock_timeout = '5s';

ALTER TABLE public.scores
  ADD COLUMN IF NOT EXISTS last_updated_at timestamptz;

COMMENT ON COLUMN public.scores.last_updated_at IS
  'Last in-app save or confirm of this answer (ztmf-misc#426). Written only by Score.Save and Score.Confirm; NULL for carried-forward and imported rows. Backfilled by 0064 from events created/updated.';
`,
		`ALTER TABLE public.scores DROP COLUMN IF EXISTS last_updated_at;`,
	)
}
