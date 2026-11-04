package migrations

// 0068 and 0069 were left free for in-flight work when this was written. tern
// numbers migrations by registration (file name) order, so a gap is harmless.
func init() {
	appendMigration(
		"store fiscal_year and cadence on datacalls",
		`
-- A data call's fiscal year and cadence used to live only in its name
-- ("FY## ZTM", "FY## Q#"), and readers parsed them out. Store them as columns
-- so the name can be a free-text title.
--
-- Backfill rule, applied to the trimmed, upper-cased name:
--   fiscal_year: a leading FY + 2 or 4 digits not followed by another digit.
--                Two digits expand to 20##. The result must be 2000-2999.
--   cadence:     a standalone Q1-Q4 token means quarterly; anything else is
--                annual.
-- A name with no usable year fails the migration and lists the rows instead of
-- guessing. Rename them to start with FY## or FY#### and redeploy.
--
-- cadence is text + CHECK rather than a native ENUM, matching 0046/0048. Its
-- DEFAULT 'annual' matches the API default on create. fiscal_year has no
-- default: every write path has to say which year a call belongs to.
--
-- Guarded so a retry is a no-op: the columns use IF NOT EXISTS, the backfill
-- only touches rows whose fiscal_year is still NULL, and the constraints are
-- added inside a DO block because Postgres has no ADD CONSTRAINT IF NOT EXISTS.
SET LOCAL lock_timeout = '10s';

ALTER TABLE public.datacalls
    ADD COLUMN IF NOT EXISTS fiscal_year integer,
    ADD COLUMN IF NOT EXISTS cadence text NOT NULL DEFAULT 'annual';

-- Rows that do not parse are left NULL here; the check below lists them and
-- aborts, which rolls this UPDATE back with the rest of the migration.
UPDATE public.datacalls d
   SET fiscal_year = CASE WHEN length(p.y) = 2 THEN 2000 + p.y::int ELSE p.y::int END,
       cadence = CASE WHEN p.n ~ '\mQ[1-4]\M' THEN 'quarterly' ELSE 'annual' END
  FROM (SELECT datacallid,
               UPPER(TRIM(datacall)) AS n,
               substring(UPPER(TRIM(datacall)) FROM '^FY([0-9]{4}|[0-9]{2})(?:[^0-9]|$)') AS y
          FROM public.datacalls
         WHERE fiscal_year IS NULL) p
 WHERE p.datacallid = d.datacallid;

DO $$
DECLARE
    unparseable text;
BEGIN
    SELECT string_agg(format('%s %L', datacallid, datacall), ', ' ORDER BY datacallid)
      INTO unparseable
      FROM public.datacalls
     WHERE fiscal_year IS NULL
        OR fiscal_year NOT BETWEEN 2000 AND 2999;

    IF unparseable IS NOT NULL THEN
        RAISE EXCEPTION 'DATACALL_FISCAL_YEAR_BACKFILL: cannot read a fiscal year from these data call names (datacallid name): %. Rename them to start with FY## or FY#### and redeploy.', unparseable;
    END IF;
END $$;

ALTER TABLE public.datacalls ALTER COLUMN fiscal_year SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'datacalls_fiscal_year_check'
          AND conrelid = 'public.datacalls'::regclass
    ) THEN
        ALTER TABLE public.datacalls
            ADD CONSTRAINT datacalls_fiscal_year_check
            CHECK (fiscal_year BETWEEN 2000 AND 2999);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'datacalls_cadence_check'
          AND conrelid = 'public.datacalls'::regclass
    ) THEN
        ALTER TABLE public.datacalls
            ADD CONSTRAINT datacalls_cadence_check
            CHECK (cadence IN ('annual', 'quarterly'));
    END IF;
END $$;
`,
		`
ALTER TABLE public.datacalls
    DROP COLUMN IF EXISTS cadence,
    DROP COLUMN IF EXISTS fiscal_year;
`,
	)
}
