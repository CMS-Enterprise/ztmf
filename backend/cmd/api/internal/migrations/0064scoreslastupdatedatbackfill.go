package migrations

func init() {
	appendMigration(
		"backfill scores.last_updated_at from score edit events",
		`
-- ztmf-misc#426. Same predicate as 0048's status backfill: only in-app edits
-- ('created'/'updated') count, so imported provenance stays NULL. Matches the
-- value the old progress lateral computed per row. Row locks only, so reads of
-- scores are not blocked while it runs.

UPDATE public.scores s
   SET last_updated_at = e.max_at
  FROM (SELECT (payload->>'scoreid')::int AS scoreid, MAX(createdat) AS max_at
          FROM public.events
         WHERE resource = 'public.scores'
           AND action IN ('created', 'updated')
         GROUP BY 1) e
 WHERE e.scoreid = s.scoreid
   AND s.last_updated_at IS DISTINCT FROM e.max_at;
`,
		`UPDATE public.scores SET last_updated_at = NULL WHERE last_updated_at IS NOT NULL;`,
	)
}
