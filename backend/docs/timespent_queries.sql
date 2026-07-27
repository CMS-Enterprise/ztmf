-- ============================================================================
-- ZTMF time-spent queries (issue #368)
-- ============================================================================
-- Rough measures of how long people spend working a Data Call, derived from the
-- events audit log. These are the canonical queries; run them directly against
-- Postgres today, or port to Snowflake once events land there.
--
-- Editor time is the primary metric (per the issue owner); viewer time is
-- reported alongside but kept out of the headline averages so the measured
-- numbers stay comparable with the editor-only historical proxy.
--
-- Two modes:
--   * MEASURED  - uses the 'viewed' events the questionnaire records now. Dwell
--                 on a question = time from OPENING it to opening the next
--                 question, capped at 30 minutes. Splits editor vs viewer via
--                 the (server-derived) readonly flag. Use for data calls that
--                 have view events (this cycle forward).
--   * PROXY     - for older data calls that predate view tracking: no 'viewed'
--                 events exist, so effort is proxied from score SAVES. A save
--                 historically meant "finished this question, clicked Next", so
--                 the gap PRECEDING a save is the time spent on THAT save's
--                 question. Editor-only, capped at 30 minutes. Use for
--                 year-over-year comparison.
--
-- Parameter: replace :datacallid with the target data call id. For the
-- per-question queries, optionally filter to one system with the commented
-- WHERE fismasystemid line.
--
-- Caveats (all "rough is fine", per #368):
--   * Every interval is capped at 30 minutes (the app session timeout), so a
--     walk-away/logout cannot inflate a total.
--   * Each mode drops the one interval it cannot bound: MEASURED drops a
--     person's LAST question (no following view); PROXY drops a person's FIRST
--     question (no preceding save). Lower bound, by design.
--   * PROXY vs MEASURED are directional, not identical: historically the app
--     saved on every "Next", whereas it now saves only on a real change, and
--     PROXY has no viewer time.
--   * Filtering: the existence joins drop views/saves whose question (MEASURED)
--     or score (PROXY) no longer exists. The system id is taken from the event
--     as-is -- to exclude invented or non-participating systems, anchor on the
--     fismasystems table (see the M1b variant). These queries do NOT verify
--     that a question applies to the submitted system's environment.
--
-- Snowflake porting notes: replace Postgres JSON access `payload->>'k'` with
-- `payload:k::string`, `EXTRACT(EPOCH FROM x)` with `DATEDIFF('second', a, b)`
-- on the raw timestamps, and `INTERVAL '30 minutes'` with an explicit
-- 1800-second bound. LEAD/LAG/LEAST and FILTER carry over unchanged.
--
-- Running: substitute :datacallid, e.g.
--   psql -v datacallid=5 -f backend/docs/timespent_queries.sql
-- (GUI clients such as DBeaver treat :datacallid as a bind parameter and
-- prompt for it; for a plain paste, replace :datacallid with the number.)
-- If these get slow on a large events table, see the optional index at the
-- bottom of this file.
-- ============================================================================


-- ============================================================================
-- MEASURED MODE (view-based; this cycle forward)
-- ============================================================================

-- M1. Editor time per system (viewer shown alongside; average is editor-only).
WITH views AS (
    SELECT e.userid,
           (e.payload->>'fismasystemid')::int AS fismasystemid,
           (e.payload->>'questionid')::int    AS questionid,
           COALESCE((e.payload->>'readonly')::boolean, FALSE) AS readonly,
           e.createdat,
           LEAD(e.createdat) OVER (
               PARTITION BY e.userid, (e.payload->>'fismasystemid')::int
               ORDER BY e.createdat
           ) AS next_at
    FROM events e
    WHERE e.resource = 'questionnaire'
      AND e.action = 'viewed'
      AND (e.payload->>'datacallid')::int = :datacallid
),
dwell AS (
    -- Only 'viewed' events bound a view (a save never truncates another
    -- question's dwell), and the question must actually exist.
    SELECT v.fismasystemid, v.userid, v.questionid, v.readonly,
           EXTRACT(EPOCH FROM LEAST(v.next_at - v.createdat, INTERVAL '30 minutes')) AS secs
    FROM views v
    JOIN questions q ON q.questionid = v.questionid
    WHERE v.next_at IS NOT NULL
)
SELECT d.fismasystemid,
       ROUND(SUM(d.secs) FILTER (WHERE NOT d.readonly))              AS editor_seconds,
       ROUND(SUM(d.secs) FILTER (WHERE d.readonly))                  AS viewer_seconds,
       COUNT(DISTINCT d.questionid) FILTER (WHERE NOT d.readonly)    AS editor_questions,
       COALESCE(ROUND(
           SUM(d.secs) FILTER (WHERE NOT d.readonly)
           / NULLIF(COUNT(DISTINCT d.questionid) FILTER (WHERE NOT d.readonly), 0)
       ), 0) AS avg_editor_seconds_per_question
FROM dwell d
GROUP BY d.fismasystemid
ORDER BY d.fismasystemid;


-- M1b (optional). Same as M1 but lists EVERY active system, including those
-- with no recorded activity (zeros), by anchoring on the fismasystems table
-- (which also drops invented system ids).
WITH views AS (
    SELECT e.userid,
           (e.payload->>'fismasystemid')::int AS fismasystemid,
           (e.payload->>'questionid')::int    AS questionid,
           COALESCE((e.payload->>'readonly')::boolean, FALSE) AS readonly,
           e.createdat,
           LEAD(e.createdat) OVER (
               PARTITION BY e.userid, (e.payload->>'fismasystemid')::int
               ORDER BY e.createdat
           ) AS next_at
    FROM events e
    WHERE e.resource = 'questionnaire'
      AND e.action = 'viewed'
      AND (e.payload->>'datacallid')::int = :datacallid
),
dwell AS (
    SELECT v.fismasystemid, v.userid, v.questionid, v.readonly,
           EXTRACT(EPOCH FROM LEAST(v.next_at - v.createdat, INTERVAL '30 minutes')) AS secs
    FROM views v
    JOIN questions q ON q.questionid = v.questionid
    WHERE v.next_at IS NOT NULL
),
per_system AS (
    SELECT d.fismasystemid,
           SUM(d.secs) FILTER (WHERE NOT d.readonly)                 AS editor_seconds,
           SUM(d.secs) FILTER (WHERE d.readonly)                     AS viewer_seconds,
           COUNT(DISTINCT d.questionid) FILTER (WHERE NOT d.readonly) AS editor_questions
    FROM dwell d
    GROUP BY d.fismasystemid
)
SELECT fs.fismasystemid,
       fs.fismaacronym,
       COALESCE(ROUND(ps.editor_seconds), 0) AS editor_seconds,
       COALESCE(ROUND(ps.viewer_seconds), 0) AS viewer_seconds,
       COALESCE(ps.editor_questions, 0)      AS editor_questions,
       COALESCE(ROUND(ps.editor_seconds / NULLIF(ps.editor_questions, 0)), 0) AS avg_editor_seconds_per_question
FROM fismasystems fs
LEFT JOIN per_system ps ON ps.fismasystemid = fs.fismasystemid
WHERE fs.decommissioned = FALSE   -- drop this line to include decommissioned systems
ORDER BY fs.fismasystemid;


-- M2. Time per person, per system (editor and viewer seconds shown separately).
WITH views AS (
    SELECT e.userid,
           (e.payload->>'fismasystemid')::int AS fismasystemid,
           (e.payload->>'questionid')::int    AS questionid,
           COALESCE((e.payload->>'readonly')::boolean, FALSE) AS readonly,
           e.createdat,
           LEAD(e.createdat) OVER (
               PARTITION BY e.userid, (e.payload->>'fismasystemid')::int
               ORDER BY e.createdat
           ) AS next_at
    FROM events e
    WHERE e.resource = 'questionnaire'
      AND e.action = 'viewed'
      AND (e.payload->>'datacallid')::int = :datacallid
),
dwell AS (
    SELECT v.fismasystemid, v.userid, v.questionid, v.readonly,
           EXTRACT(EPOCH FROM LEAST(v.next_at - v.createdat, INTERVAL '30 minutes')) AS secs
    FROM views v
    JOIN questions q ON q.questionid = v.questionid
    WHERE v.next_at IS NOT NULL
)
SELECT d.fismasystemid,
       u.fullname, u.email, u.role,
       ROUND(SUM(d.secs) FILTER (WHERE NOT d.readonly))              AS editor_seconds,
       ROUND(SUM(d.secs) FILTER (WHERE d.readonly))                  AS viewer_seconds,
       COUNT(DISTINCT d.questionid) FILTER (WHERE NOT d.readonly)    AS editor_questions
FROM dwell d
LEFT JOIN users u ON u.userid = d.userid
GROUP BY d.fismasystemid, u.fullname, u.email, u.role
ORDER BY d.fismasystemid, editor_seconds DESC NULLS LAST;


-- M3. Average editor time per question, within a system (per-person average;
-- viewer seconds shown alongside). Uncomment the WHERE to focus on one system.
WITH views AS (
    SELECT e.userid,
           (e.payload->>'fismasystemid')::int AS fismasystemid,
           (e.payload->>'questionid')::int    AS questionid,
           COALESCE((e.payload->>'readonly')::boolean, FALSE) AS readonly,
           e.createdat,
           LEAD(e.createdat) OVER (
               PARTITION BY e.userid, (e.payload->>'fismasystemid')::int
               ORDER BY e.createdat
           ) AS next_at
    FROM events e
    WHERE e.resource = 'questionnaire'
      AND e.action = 'viewed'
      AND (e.payload->>'datacallid')::int = :datacallid
),
dwell AS (
    SELECT v.fismasystemid, v.userid, v.questionid, v.readonly,
           EXTRACT(EPOCH FROM LEAST(v.next_at - v.createdat, INTERVAL '30 minutes')) AS secs
    FROM views v
    WHERE v.next_at IS NOT NULL
)
SELECT d.fismasystemid,
       d.questionid,
       q.question,
       COUNT(DISTINCT d.userid) FILTER (WHERE NOT d.readonly)        AS editor_people,
       COALESCE(ROUND(
           SUM(d.secs) FILTER (WHERE NOT d.readonly)
           / NULLIF(COUNT(DISTINCT d.userid) FILTER (WHERE NOT d.readonly), 0)
       ), 0) AS avg_editor_seconds_per_person,
       ROUND(SUM(d.secs) FILTER (WHERE d.readonly))                  AS viewer_seconds
FROM dwell d
JOIN questions q ON q.questionid = d.questionid
-- WHERE d.fismasystemid = <id>
GROUP BY d.fismasystemid, d.questionid, q.question
ORDER BY d.fismasystemid, d.questionid;


-- ============================================================================
-- PROXY MODE (save-gap; older data calls, editor-only, for year-over-year)
-- ============================================================================
-- A save marked "finished this question, clicked Next", so the gap PRECEDING a
-- save (LAG) is the time spent on that save's OWN question. The first save per
-- (user, system) has no preceding save and is dropped.

-- P1. Editor time per system.
WITH saves AS (
    SELECT e.userid,
           (e.payload->>'fismasystemid')::int AS fismasystemid,
           (e.payload->>'scoreid')::int       AS scoreid,
           e.createdat,
           LAG(e.createdat) OVER (
               PARTITION BY e.userid, (e.payload->>'fismasystemid')::int
               ORDER BY e.createdat
           ) AS prev_at
    FROM events e
    WHERE e.resource = 'public.scores'
      AND e.action IN ('created', 'updated')   -- exclude bulk-import provenance
      AND (e.payload->>'datacallid')::int = :datacallid
),
dwell AS (
    -- Attribute the gap before each save to THAT save's question; the join also
    -- drops any save whose score no longer exists.
    SELECT s.fismasystemid, s.userid, f.questionid,
           EXTRACT(EPOCH FROM LEAST(s.createdat - s.prev_at, INTERVAL '30 minutes')) AS secs
    FROM saves s
    JOIN scores sc          ON sc.scoreid = s.scoreid
    JOIN functionoptions fo ON fo.functionoptionid = sc.functionoptionid
    JOIN functions f        ON f.functionid = fo.functionid
    WHERE s.prev_at IS NOT NULL
)
SELECT d.fismasystemid,
       ROUND(SUM(d.secs))                                           AS editor_seconds,
       COUNT(DISTINCT d.questionid)                                 AS editor_questions,
       COALESCE(ROUND(SUM(d.secs) / NULLIF(COUNT(DISTINCT d.questionid), 0)), 0) AS avg_editor_seconds_per_question
FROM dwell d
GROUP BY d.fismasystemid
ORDER BY d.fismasystemid;


-- P2. Time per person, per system.
WITH saves AS (
    SELECT e.userid,
           (e.payload->>'fismasystemid')::int AS fismasystemid,
           (e.payload->>'scoreid')::int       AS scoreid,
           e.createdat,
           LAG(e.createdat) OVER (
               PARTITION BY e.userid, (e.payload->>'fismasystemid')::int
               ORDER BY e.createdat
           ) AS prev_at
    FROM events e
    WHERE e.resource = 'public.scores'
      AND e.action IN ('created', 'updated')
      AND (e.payload->>'datacallid')::int = :datacallid
),
dwell AS (
    SELECT s.fismasystemid, s.userid, f.questionid,
           EXTRACT(EPOCH FROM LEAST(s.createdat - s.prev_at, INTERVAL '30 minutes')) AS secs
    FROM saves s
    JOIN scores sc          ON sc.scoreid = s.scoreid
    JOIN functionoptions fo ON fo.functionoptionid = sc.functionoptionid
    JOIN functions f        ON f.functionid = fo.functionid
    WHERE s.prev_at IS NOT NULL
)
SELECT d.fismasystemid,
       u.fullname, u.email, u.role,
       ROUND(SUM(d.secs))           AS editor_seconds,
       COUNT(DISTINCT d.questionid) AS editor_questions
FROM dwell d
LEFT JOIN users u ON u.userid = d.userid
GROUP BY d.fismasystemid, u.fullname, u.email, u.role
ORDER BY d.fismasystemid, editor_seconds DESC NULLS LAST;


-- P3. Average editor time per question, within a system.
WITH saves AS (
    SELECT e.userid,
           (e.payload->>'fismasystemid')::int AS fismasystemid,
           (e.payload->>'scoreid')::int       AS scoreid,
           e.createdat,
           LAG(e.createdat) OVER (
               PARTITION BY e.userid, (e.payload->>'fismasystemid')::int
               ORDER BY e.createdat
           ) AS prev_at
    FROM events e
    WHERE e.resource = 'public.scores'
      AND e.action IN ('created', 'updated')
      AND (e.payload->>'datacallid')::int = :datacallid
),
dwell AS (
    SELECT s.fismasystemid, s.userid, f.questionid,
           EXTRACT(EPOCH FROM LEAST(s.createdat - s.prev_at, INTERVAL '30 minutes')) AS secs
    FROM saves s
    JOIN scores sc          ON sc.scoreid = s.scoreid
    JOIN functionoptions fo ON fo.functionoptionid = sc.functionoptionid
    JOIN functions f        ON f.functionid = fo.functionid
    WHERE s.prev_at IS NOT NULL
)
SELECT d.fismasystemid,
       d.questionid,
       q.question,
       COUNT(DISTINCT d.userid)                              AS editor_people,
       COALESCE(ROUND(SUM(d.secs) / NULLIF(COUNT(DISTINCT d.userid), 0)), 0) AS avg_editor_seconds_per_person
FROM dwell d
JOIN questions q ON q.questionid = d.questionid
-- WHERE d.fismasystemid = <id>
GROUP BY d.fismasystemid, d.questionid, q.question
ORDER BY d.fismasystemid, d.questionid;


-- ============================================================================
-- OPTIONAL: performance index
-- ============================================================================
-- These queries scan the append-only events table filtered by data call and
-- windowed per (userid, fismasystemid). For occasional manual runs that is
-- fine. If they get slow on a large events table, this partial expression index
-- makes them an index-ordered read. It is intentionally NOT a migration: it
-- taxes every event write (saves + view pings) to speed up rarely-run
-- analytics, so only add it if the read cost actually justifies that trade.
--
-- CREATE INDEX IF NOT EXISTS events_timespent_idx
--     ON public.events (
--         (((payload->>'datacallid')::int)),
--         userid,
--         (((payload->>'fismasystemid')::int)),
--         createdat
--     )
--     WHERE resource IN ('questionnaire', 'public.scores');
