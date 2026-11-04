package migrations

// The 0070 in the file name is not the version a database records. tern
// numbers a migration by its position in the registry, which is file name
// order, so on a database that holds every migration before this one, this is
// recorded as version 68. The 0068 and 0069 file names were left free for the
// questionnaire-version migrations that were still open when this was written.
// Whichever of those and this lands second must rename its files to sort after
// the other's, or a database that already ran the first would record the wrong
// migration under each version and skip one.
func init() {
	appendMigration(
		"store fiscal_year and cadence on datacalls; titles unique ignoring case",
		`
-- A data call's fiscal year and cadence used to live only in its name
-- ("FY## ZTM", "FY## Q#"), and readers parsed them out. Store them as columns
-- so the name can be a free-text title.
--
-- Free-text titles are no longer upper-cased by the UI, and the UI resolves
-- questionnaire deep links by title case-insensitively. Two titles that differ
-- only by case would make a link open the wrong call, so titles are unique on
-- lower(datacall). datacall_key (exact-match UNIQUE) stays as it is.
--
-- Backfill rule, applied to the trimmed, upper-cased name:
--   fiscal_year: a leading FY + 2 or 4 digits not followed by another digit.
--                Two digits expand to 20##. The result must be 2000-2999.
--   cadence:     quarterly when Q1-Q4 comes right after the year, with or
--                without whitespace, and ends a word ("FY26 Q2", "FY26Q2").
--                Anything else is annual, including a quarter later in the
--                name ("FY2027 Zero Trust Q2 Review"). This matches the UI's
--                name rule.
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
       cadence = CASE WHEN p.n ~ '^FY([0-9]{4}|[0-9]{2})\s*Q[1-4]\M' THEN 'quarterly' ELSE 'annual' END
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

-- Existing titles that differ only by case would make the index build fail
-- with a bare 23505 naming one pair. List every such row instead.
DO $$
DECLARE
    dupes text;
BEGIN
    SELECT string_agg(format('%s %L', datacallid, datacall), ', ' ORDER BY lower(datacall), datacallid)
      INTO dupes
      FROM public.datacalls
     WHERE lower(datacall) IN (SELECT lower(datacall)
                                 FROM public.datacalls
                                GROUP BY 1
                               HAVING count(*) > 1);

    IF dupes IS NOT NULL THEN
        RAISE EXCEPTION 'DATACALL_TITLE_CASE_DUPLICATES: these data call titles differ only by case (datacallid name): %. Rename them so no two match ignoring case and redeploy.', dupes;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS datacalls_datacall_lower_key
    ON public.datacalls (lower(datacall));
`,
		`
DROP INDEX IF EXISTS public.datacalls_datacall_lower_key;

ALTER TABLE public.datacalls
    DROP COLUMN IF EXISTS cadence,
    DROP COLUMN IF EXISTS fiscal_year;
`,
	)
}
