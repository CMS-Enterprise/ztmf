package migrations

func init() {
	appendMigration(
		"denormalize functionid onto scores with a maintaining trigger and composite FK",
		`
-- ztmf#491. An answer's natural key is (fismasystemid, datacallid, question),
-- but the question is reachable only via functionoptions.functionid and
-- Postgres cannot index across that hop. Copying functionid onto scores makes
-- the key indexable (0061).
SET LOCAL lock_timeout = '10s';

-- Fail readably instead of on a bare 23502 three statements down.
DO $$
DECLARE orphans bigint;
BEGIN
    SELECT count(*) INTO orphans
      FROM public.scores s
      LEFT JOIN public.functionoptions fo ON fo.functionoptionid = s.functionoptionid
     WHERE fo.functionoptionid IS NULL;
    IF orphans > 0 THEN
        RAISE EXCEPTION 'ZTMF491_ORPHANED_SCORES: % scores rows name a functionoptionid that does not exist, so functionid cannot be derived. They must be resolved in this database before this image can start against it (ztmf#491).', orphans;
    END IF;
END $$;

ALTER TABLE public.scores
  ADD COLUMN IF NOT EXISTS functionid integer;

UPDATE public.scores s
   SET functionid = fo.functionid
  FROM public.functionoptions fo
 WHERE fo.functionoptionid = s.functionoptionid
   AND s.functionid IS DISTINCT FROM fo.functionid;

CREATE OR REPLACE FUNCTION public.scores_set_functionid()
RETURNS TRIGGER AS $$
BEGIN
    SELECT fo.functionid INTO NEW.functionid
      FROM public.functionoptions fo
     WHERE fo.functionoptionid = NEW.functionoptionid;

    -- Raise 23503, not 23502. NOT NULL is checked at tuple insertion, before
    -- the AFTER-trigger FK fires, and trapError has no 23502 case - so without
    -- this a bad functionoptionid returns 500 where it returns 400 today.
    IF NOT FOUND THEN
        RAISE EXCEPTION 'scores.functionoptionid % does not exist in functionoptions',
                        NEW.functionoptionid USING ERRCODE = 'foreign_key_violation';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_scores_set_functionid ON public.scores;
CREATE TRIGGER trg_scores_set_functionid
    BEFORE INSERT OR UPDATE OF functionoptionid, functionid ON public.scores
    FOR EACH ROW
    EXECUTE FUNCTION public.scores_set_functionid();

-- NOT NULL is the constraint, not tidiness: a btree unique index treats NULLs
-- as distinct, so one NULL functionid would let 0061 build and enforce nothing
-- for that row. Must stay atomic with the backfill above.
ALTER TABLE public.scores
  ALTER COLUMN functionid SET NOT NULL;

-- Covers the gap the trigger structurally cannot: the trigger fires only on
-- writes to scores and is blind to writes to functionoptions, which is edited
-- by hand out of band. NO ACTION blocks repointing an option at a different
-- function behind the trigger's back.
CREATE UNIQUE INDEX IF NOT EXISTS functionoptions_functionoptionid_functionid_uniq
    ON public.functionoptions (functionoptionid, functionid);

ALTER TABLE public.scores
  DROP CONSTRAINT IF EXISTS scores_functionoption_function_fkey;
ALTER TABLE public.scores
  ADD CONSTRAINT scores_functionoption_function_fkey
  FOREIGN KEY (functionoptionid, functionid)
  REFERENCES public.functionoptions (functionoptionid, functionid);

COMMENT ON COLUMN public.scores.functionid IS
  'Denormalized from functionoptions.functionid (ztmf#491). Maintained by trg_scores_set_functionid and pinned by scores_functionoption_function_fkey; never write it directly.';
		`,
		`
ALTER TABLE public.scores DROP CONSTRAINT IF EXISTS scores_functionoption_function_fkey;
DROP INDEX IF EXISTS public.functionoptions_functionoptionid_functionid_uniq;
DROP TRIGGER IF EXISTS trg_scores_set_functionid ON public.scores;
DROP FUNCTION IF EXISTS public.scores_set_functionid();
ALTER TABLE public.scores DROP COLUMN IF EXISTS functionid;
		`)
}
