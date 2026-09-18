package migrations

func init() {
	appendMigration(
		"add append-only score_revisions for answer history and undo",
		`
-- ztmf-misc#391. An append-only record of both sides of every answer change,
-- written in the SAME transaction as the score write so a revision cannot exist
-- without the write it records, or the write without its revision.
--
-- events cannot serve this. Its payload is the RETURNING row, so it carries the
-- state AFTER the write and no "before" side at all; copyPreviousScores writes
-- the rollover through a raw conn.Exec precisely so it records nothing, leaving
-- every carried-forward row with zero events - and the first edit of a
-- carried-forward answer is the most common undo in the product.
--
-- No backfill. Synthesising history from events yields a wrong prev for the
-- first edit of every score, and a history that lets a user undo to a value
-- that never existed is worse than no history.
--
-- NOTE (PG->Snowflake): this table is NOT in the ztmf-insights data-sync
-- allowlist (backend/cmd/data-sync/internal/sync/sync.go, allTables) and so does
-- NOT replicate. Deliberate - it holds a second attributed copy of every
-- narrative justification. If it is ever added it needs the same
-- sdl_sync_enabled predicate the scores entry carries; fismasystemid is
-- denormalised below precisely so that filter applies without a join.
SET LOCAL lock_timeout = '10s';

CREATE TABLE IF NOT EXISTS public.score_revisions (
    revisionid  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scoreid     INT NOT NULL REFERENCES public.scores (scoreid) ON DELETE CASCADE,
    revision_no INT NOT NULL,

    -- Denormalised so a read is OpDiv-scopable without joining scores. This is
    -- the fix for the reason GET /events had to be restricted to
    -- HasUnscopedRead(): events carries no opdiv_id to scope on, so an ISSO
    -- cannot read its own system's audit trail at all.
    --
    -- datacallid cascades in its own right rather than relying on cascade
    -- ordering through scores (scores.datacallid is already ON DELETE CASCADE,
    -- migration 0007). Two independent cascade paths in one DELETE is not
    -- something to stake a production rollback on.
    fismasystemid INT NOT NULL REFERENCES public.fismasystems (fismasystemid),
    datacallid    INT NOT NULL REFERENCES public.datacalls (datacallid) ON DELETE CASCADE,

    -- The question this revision is about, derived from
    -- functionoptions.functionid at write time. It makes a revision
    -- self-describing rather than only row-identified: once ztmf-misc#397 can
    -- re-point a cycle at a different questionnaire version, the stored
    -- functionoptionid lives in the OLD version's namespace and the whole
    -- pre-bump history becomes unreadable without this column.
    --
    -- Derived from functionoptions rather than copied from scores.functionid,
    -- which does not exist on main (it arrives with ztmf#491 / ztmf#594). That
    -- keeps this migration applicable either way.
    functionid INT NOT NULL REFERENCES public.functions (functionid),

    -- 'translate' is unused today and present deliberately: ztmf-misc#397 writes
    -- one per row it rewrites during a version re-pin. Widening a CHECK later is
    -- an ALTER on a table this design otherwise treats as append-only, so the
    -- value is cheaper to reserve now than to add then.
    kind varchar(16) NOT NULL
        CONSTRAINT score_revisions_kind_check
        CHECK (kind IN ('create', 'update', 'confirm', 'undo', 'translate')),

    -- prev_* is NULL as a set exactly when kind = 'create': there is no earlier
    -- value, which is also why a create is never undoable - undo moves between
    -- values that existed and there is no path back to unanswered.
    prev_functionoptionid    INT,
    prev_notes               varchar(2000),
    prev_notes_is_ai_summary BOOLEAN,
    prev_status              varchar(20),

    new_functionoptionid     INT     NOT NULL,
    new_notes                varchar(2000),
    new_notes_is_ai_summary  BOOLEAN NOT NULL,
    new_status               varchar(20) NOT NULL,

    -- Undo APPENDS a revision rather than popping the head, so who undid what
    -- stays auditable and redo is just undoing the undo.
    undoes_revisionid BIGINT REFERENCES public.score_revisions (revisionid),

    userid    uuid NOT NULL REFERENCES public.users (userid),
    createdat TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- The backstop, not the mechanism. revision_no is allocated as MAX+1 while
    -- the writer holds the score row's FOR UPDATE lock, which is what makes two
    -- concurrent Saves land 1 and 2 rather than one of them surfacing a 23505.
    CONSTRAINT score_revisions_scoreid_revision_no_uniq UNIQUE (scoreid, revision_no),

    -- Biconditional on purpose. "create implies no prev" alone would still let
    -- a non-create row carry a NULL prev_functionoptionid, and undo reads
    -- head.prev unconditionally once it has ruled out a create - so a row the
    -- app never writes would be a nil dereference rather than a clean refusal.
    CONSTRAINT score_revisions_prev_iff_not_create
        CHECK ((kind = 'create') = (prev_functionoptionid IS NULL)),
    CONSTRAINT score_revisions_undo_names_target
        CHECK ((kind = 'undo') = (undoes_revisionid IS NOT NULL))
);

-- Deliberately NO foreign key on prev_functionoptionid / new_functionoptionid.
-- A revision is an immutable record of what was chosen at a point in time. A
-- catalog row disappearing - reachable through the admin endpoints today, and a
-- first-class operation on draft versions once ztmf-misc#396 lands - must
-- neither cascade history away nor block the delete. Reads LEFT JOIN
-- functionoptions for labels and tolerate a NULL.
COMMENT ON COLUMN public.score_revisions.prev_functionoptionid IS
  'Intentionally unconstrained: history must survive deletion of the option it names (ztmf-misc#391).';

-- The history read: one score, newest first.
CREATE INDEX IF NOT EXISTS score_revisions_score_idx
    ON public.score_revisions (scoreid, revision_no DESC);

-- The OpDiv-scopable read the denormalised columns exist for.
CREATE INDEX IF NOT EXISTS score_revisions_scope_idx
    ON public.score_revisions (fismasystemid, datacallid);

COMMENT ON TABLE public.score_revisions IS
  'Append-only answer history (ztmf-misc#391). Written in the score write transaction; never updated, only inserted. Deleting a data call destroys that cycle history by design.';
		`,
		`
DROP TABLE IF EXISTS public.score_revisions;
		`)
}
