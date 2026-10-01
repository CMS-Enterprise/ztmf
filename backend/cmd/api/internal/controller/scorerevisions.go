package controller

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// undoRequest is the undo body. expected_head_revisionid is a pointer so a
// missing field is distinguishable from a zero one, and is required: an undo
// without a concurrency token is last-write-wins on a destructive action, and
// the client always holds the head from the history it just rendered.
type undoRequest struct {
	ExpectedHeadRevisionID *int64 `json:"expected_head_revisionid"`
}

//	@Summary	Read an answer's revision history
//	@Tags		scores
//	@Produce	json
//	@Security	bearerAuth
//	@Param		scoreid	path		int	true	"Score ID"
//	@Success	200		{object}	apiResponse[model.ScoreHistory]
//	@Failure	403		{object}	apiResponse[any]
//	@Failure	404		{object}	apiResponse[any]
//	@Failure	500		{object}	apiResponse[any]
//	@Router		/scores/{scoreid}/revisions [get]
func ListScoreRevisions(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	var scoreID int32
	fmt.Sscan(mux.Vars(r)["scoreid"], &scoreID)

	// No role-only pre-check here, unlike the write paths: every authenticated
	// tier may read SOME history, so there is no tier to reject before the
	// lookup. The row is loaded first and authorization runs against its own
	// fismasystemid, so a client asserts nothing but the scoreid.
	score, err := model.FindScoreByID(r.Context(), scoreID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// guardViewFismaSystem rather than a new read guard: it already is
	// "guardScoreWrite but admitting read-only tiers", short-circuits without a
	// DB hit for unscoped-read and assigned callers, and returns NotFound for a
	// missing system. A near-duplicate is the drift guardScoreWrite was
	// extracted to prevent.
	if err := guardViewFismaSystem(r.Context(), user, score.FismaSystemID); err != nil {
		respond(w, r, nil, err)
		return
	}

	// Undoability is reader-relative, so the policy is computed from the same
	// guard the undo endpoint enforces. A read-only admin gets history with
	// every row marked not undoable and a reason, which is what makes the
	// drawer viewable for audit without the client knowing the role matrix.
	history, err := model.FindScoreRevisions(r.Context(), score, model.ScoreUndoPolicy{
		CanWrite: guardScoreWrite(r.Context(), user, score.FismaSystemID) == nil,
	})

	respond(w, r, history, err)
}

//	@Summary	Undo the most recent change to an answer
//	@Tags		scores
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		scoreid	path		int					true	"Score ID"
//	@Param		body	body		undoRequest			true	"Head revision the caller expects"
//	@Success	200		{object}	apiResponse[model.ScoreUndoResult]
//	@Failure	400		{object}	apiResponse[any]
//	@Failure	403		{object}	apiResponse[any]
//	@Failure	404		{object}	apiResponse[any]
//	@Failure	409		{object}	apiResponse[any]
//	@Failure	500		{object}	apiResponse[any]
//	@Router		/scores/{scoreid}/revisions/undo [post]
func UndoScoreRevision(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	// Role-only rejection before any DB access, matching SaveScore and
	// ConfirmScore and preserving the property pinned in
	// rbac_enforcement_test.go. Without it a read-only tier could tell an
	// existing scoreid from a missing one by the 403-vs-404 difference.
	if user.IsReadOnlyAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	// Body before the lookup, for the same non-disclosure reason: a malformed
	// request is a 400 whether or not the score exists.
	body := undoRequest{}
	if err := getJSON(r.Body, &body); err != nil {
		respond(w, r, nil, ErrMalformed)
		return
	}
	var scoreID int32
	fmt.Sscan(mux.Vars(r)["scoreid"], &scoreID)

	score, err := model.FindScoreByID(r.Context(), scoreID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// guardScoreWrite unchanged, and deliberately no IsSystemDelegate rejection:
	// undo is a data-call answer surface, so an assigned delegate may use it.
	// See TestSystemDelegate_AllowedAnswerSurfaces.
	if err := guardScoreWrite(r.Context(), user, score.FismaSystemID); err != nil {
		respond(w, r, nil, err)
		return
	}

	// The required-field check lives in the model alongside the rest of the undo
	// rules, so a missing token is a 400 field map rather than a bare 400.
	result, err := model.UndoScoreRevision(r.Context(), score, body.ExpectedHeadRevisionID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// respondOK, not respond: undo is a state transition on an existing answer,
	// and respond() would emit 201 for a POST. Same convention ConfirmScore uses.
	respondOK(w, result)
}
