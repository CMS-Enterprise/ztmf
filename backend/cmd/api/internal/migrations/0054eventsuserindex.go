package migrations

func init() {
	appendMigration(
		"add per-user index on events for last-seen lookups",
		`
-- The users list derives last_seen as MAX(events.createdat) per user, one
-- correlated subquery per row, the same shape the existing assignedopdivids
-- subquery uses. events carries no userid index today: it has one on
-- (resource, createdat) and a partial one on the scoreid expression, neither
-- of which serves a lookup keyed on userid.
--
-- Without this the subquery is a sequential scan of events per user row. At
-- roughly 200k events and 1,250 users that is a full scan per row on a page
-- an admin loads routinely, which is exactly the regression 0037 was written
-- to avoid for the score audit lateral.
--
-- createdat DESC alongside userid so the MAX is an index-only descent to the
-- first matching entry rather than an aggregate over the whole partition.
-- Not partial: unlike the score audit index there is no single resource worth
-- narrowing to, since last_seen deliberately counts activity of any kind.
CREATE INDEX IF NOT EXISTS events_user_activity_idx
    ON public.events (userid, createdat DESC);
`,
		`DROP INDEX IF EXISTS events_user_activity_idx;`,
	)
}
