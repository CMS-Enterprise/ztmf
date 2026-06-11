package migrations

func init() {
	getMigrator().AppendMigration(
		"add index on fismasystems.opdiv_id for OpDiv scope predicates",
		`
-- OpDiv-scoped RBAC enforcement filters fismasystems by opdiv_id on every
-- scoped read (list systems, scores, datacall completion) for the OPDIV_ADMIN
-- / OPDIV_READONLY_ADMIN tiers. The column is the backfilled FK added in the
-- multi-OpDiv schema (0026-0040) and is otherwise unindexed, so each scoped
-- read degenerates to a sequential scan over fismasystems.
--
-- The predicate shapes are opdiv_id = ANY($1) (direct) and a correlated
-- subquery (scores). A plain btree on opdiv_id serves both. Cardinality is
-- low (one row per OpDiv, ~14 today) but the table is small enough that the
-- planner will still prefer the index for the single-OpDiv-admin case, and it
-- removes the seq-scan risk as the system count grows across OpDivs.
--
-- IF NOT EXISTS so the migration is idempotent on a retry. CONCURRENTLY is
-- unavailable (tern wraps each migration in a transaction); fismasystems is
-- small (~255 rows prod) so the brief lock is negligible.

CREATE INDEX IF NOT EXISTS fismasystems_opdiv_id_idx
    ON public.fismasystems (opdiv_id);
        `,
		`
DROP INDEX IF EXISTS public.fismasystems_opdiv_id_idx;
        `)
}
