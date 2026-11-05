package migrations

func init() {
	appendMigration(
		"enforce one answer per system per question per data call on scores",
		`
-- ztmf#491. The uniqueness that should always have existed: a system has at
-- most one stored answer per question per data call.
--
-- This key assumes a data call pins one questionnaire version. If a cycle ever
-- spans versions, the version column joins the index.
SET LOCAL lock_timeout = '10s';

-- Existing duplicates are reconciled out-of-band before this deploys, since a
-- pair holding two different answers needs the data owner rather than a rule.
-- Fail with a countable, greppable message rather than a bare 23505 from inside
-- a crash-looping task.
DO $$
DECLARE dupes bigint;
BEGIN
    SELECT count(*) INTO dupes
      FROM (SELECT 1
              FROM public.scores
             GROUP BY fismasystemid, datacallid, functionid
            HAVING count(*) > 1) d;
    IF dupes > 0 THEN
        RAISE EXCEPTION 'ZTMF491_DUPLICATE_ANSWERS: % (system, data call, question) groups hold more than one answer, so the index cannot be built. They must be reconciled in this database before this image can start against it (ztmf#491).', dupes;
    END IF;
END $$;

-- A bare index, not ADD CONSTRAINT UNIQUE: the rollover dedup tests must drop
-- it around their fixture, and DROP INDEX refuses on a constraint-backed index.
--
-- Not CONCURRENTLY. tern honours "---- tern: disable-tx ----" so it is
-- available, but a failed concurrent build leaves an invalid index that the
-- IF NOT EXISTS then skips forever, on a migration that re-runs every boot.
CREATE UNIQUE INDEX IF NOT EXISTS scores_fismasystem_datacall_function_uniq
    ON public.scores (fismasystemid, datacallid, functionid);

COMMENT ON INDEX public.scores_fismasystem_datacall_function_uniq IS
  'One answer per system per question per data call (ztmf#491). Unlike the trigger and FK on scores.functionid, an index is still enforced under session_replication_role = replica - which is how the duplicates it prevents were created.';
		`,
		`DROP INDEX IF EXISTS public.scores_fismasystem_datacall_function_uniq;`)
}
