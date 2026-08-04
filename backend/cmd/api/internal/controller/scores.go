package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

//	@Summary	List all scores
//	@Tags		scores
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	query		int	false	"Filter by FISMA system ID"
//	@Param		datacallid		query		int	false	"Filter by data call ID"
//	@Success	200				{object}	apiResponse[[]model.Score]
//	@Failure	400				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/scores [get]
func ListScores(w http.ResponseWriter, r *http.Request) {

	var (
		scores []*model.Score
		err    error
	)
	user := model.UserFromContext(r.Context())
	findScoresInput := model.FindScoresInput{}

	err = decoder.Decode(&findScoresInput, r.URL.Query())

	// Scope by tier AFTER decode so a client cannot widen scope via query
	// params: unscoped admins see all; OPDIV tiers fail-closed to their granted
	// OpDivs' systems; ISSO/ISSM keep the per-system (UserID) path.
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		findScoresInput.RestrictToOpDivIDs = true
		_, findScoresInput.OpDivIDs = user.EffectiveOpDivScope()
	default:
		findScoresInput.UserID = &user.UserID
	}

	if err == nil {
		scores, err = model.FindScores(r.Context(), findScoresInput)
	}

	respond(w, r, scores, err)
}

//	@Summary	Create or update a score
//	@Tags		scores
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		scoreid	path		int			false	"Score ID"
//	@Param		body	body		model.Score	true	"Score to save"
//	@Success	201		{object}	apiResponse[model.Score]
//	@Success	204		"No Content"
//	@Failure	400		{object}	apiResponse[any]
//	@Failure	403		{object}	apiResponse[any]
//	@Failure	404		{object}	apiResponse[any]
//	@Failure	500		{object}	apiResponse[any]
//	@Router		/scores [post]
//	@Router		/scores/{scoreid} [put]
func SaveScore(w http.ResponseWriter, r *http.Request) {
	var (
		scoreID int32
		err     error
	)

	user := model.UserFromContext(r.Context())
	score := &model.Score{}

	err = getJSON(r.Body, &score)
	if err != nil {
		log.Println(err)
	}

	// Role-only rejection before any DB access, matching ConfirmScore and
	// preserving the property pinned in rbac_enforcement_test.go ("read-only
	// tiers are blocked before any DB access"). Without this the update path
	// below would look the row up first, which both breaks that invariant and
	// lets a read-only caller tell an existing scoreid from a missing one by
	// the 404-vs-403 difference. guardScoreWrite re-checks this; harmless.
	if user.IsReadOnlyAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)

	if v, ok := vars["scoreid"]; ok {
		fmt.Sscan(v, &scoreID)
		score.ScoreID = scoreID
	}

	// Authorization is deliberately asymmetric between create and update.
	//
	// UPDATE: authorize against the STORED row, never the request body. The
	// body's fismasystemid is a client assertion; trusting it let any caller
	// with write access to one system reach every score row by id, since the
	// UPDATE is keyed on scoreid alone. Load the row first and guard on what
	// is actually there. The stored datacallid is also carried onto the
	// receiver so the deadline is evaluated against the cycle the row belongs
	// to rather than one the caller names.
	//
	// CREATE: no row exists yet, so the body's fismasystemid is the only
	// thing to authorize against, which is correct there.
	if score.ScoreID != 0 {
		stored, err := model.FindScoreByID(r.Context(), score.ScoreID)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		if err := guardScoreWrite(r.Context(), user, stored.FismaSystemID); err != nil {
			respond(w, r, nil, err)
			return
		}
		// Both columns are immutable on update (Save omits them from the SET
		// list); pinning them here keeps the receiver honest for the deadline
		// check and the no-op comparison.
		score.FismaSystemID = stored.FismaSystemID
		score.DataCallID = stored.DataCallID

		// Hand Save the row we just read so its no-op comparison does not
		// fetch it again. The questionnaire PUTs on every Next click, so this
		// is the hottest write path in the app.
		score, err = score.Save(r.Context(), model.WithCurrentScore(stored))
		respond(w, r, score, err)
		return
	}

	if err := guardScoreWrite(r.Context(), user, score.FismaSystemID); err != nil {
		respond(w, r, nil, err)
		return
	}

	score, err = score.Save(r.Context())

	respond(w, r, score, err)
}

// guardScoreWrite is the shared authorization for every score-mutating
// endpoint (SaveScore, ConfirmScore): read-only admins never write; ISSO/ISSM
// must hold the per-system assignment; admin tiers must manage the system's
// OpDiv (OWNER/HHS_ADMIN any, OPDIV_ADMIN only their grants). Extracted so the
// confirm path cannot drift from the save path's rules.
func guardScoreWrite(ctx context.Context, user *model.User, fismaSystemID int32) error {
	if user.IsReadOnlyAdmin() {
		return ErrForbidden
	}

	if !user.IsAdmin() && !user.IsAssignedFismaSystem(fismaSystemID) {
		return ErrForbidden
	}

	if user.IsAdmin() {
		if _, err := guardManageFismaSystem(ctx, user, fismaSystemID); err != nil {
			return err
		}
	}

	return nil
}

// ConfirmScore marks a carried-forward answer as affirmed for its data call:
// it flips scores.status to 'done' and touches nothing else. Agreement
// changes no answer field, so the ordinary PUT's no-op guard (correctly)
// refuses to record it - this is the explicit act that does.
//
// The row is loaded FIRST and authorization runs against its own
// fismasystemid - a client asserts nothing but the scoreid.
//
//	@Summary	Confirm a carried-forward score without changing it
//	@Tags		scores
//	@Produce	json
//	@Security	bearerAuth
//	@Param		scoreid	path		int	true	"Score ID"
//	@Success	200		{object}	apiResponse[model.Score]
//	@Failure	403		{object}	apiResponse[any]
//	@Failure	404		{object}	apiResponse[any]
//	@Failure	500		{object}	apiResponse[any]
//	@Router		/scores/{scoreid}/confirm [put]
func ConfirmScore(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	// Role-only rejection before any DB access, preserving the pinned
	// property from rbac_enforcement_test.go ("read-only tiers are blocked
	// before any DB access"). guardScoreWrite re-checks this; harmless.
	if user.IsReadOnlyAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	var scoreID int32
	fmt.Sscan(mux.Vars(r)["scoreid"], &scoreID)

	score, err := model.FindScoreByID(r.Context(), scoreID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	if err := guardScoreWrite(r.Context(), user, score.FismaSystemID); err != nil {
		respond(w, r, nil, err)
		return
	}

	confirmed, err := score.Confirm(r.Context())
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	// respondOK, not respond: PUT-as-action endpoints return the updated
	// entity (the restore/reactivate convention); respond() would drop the
	// body with a 204.
	respondOK(w, confirmed)
}

//	@Summary	Diff scores between two data calls
//	@Description	Compares the score (functionoption) answers of two data calls and returns only the questionnaire functions whose answer changed, each annotated with who made the later change and when. Scoped to the caller's tier: unscoped admins see all systems, OpDiv-scoped admins their OpDivs' systems, and ISSO/ISSM their assigned systems.
//	@Tags		scores
//	@Produce	json
//	@Security	bearerAuth
//	@Param		from			query		int	true	"Data call ID to compare from (earlier cycle)"
//	@Param		to				query		int	true	"Data call ID to compare to (later cycle)"
//	@Param		fismasystemid	query		int	false	"Limit the diff to a single FISMA system"
//	@Success	200				{object}	apiResponse[[]model.ScoreDiff]
//	@Failure	400				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/scores/diff [get]
func GetScoresDiff(w http.ResponseWriter, r *http.Request) {
	var (
		diffs []*model.ScoreDiff
		err   error
	)

	user := model.UserFromContext(r.Context())
	input := model.FindScoreDiffInput{}

	err = decoder.Decode(&input, r.URL.Query())

	// Same tier scoping as ListScores, applied AFTER decode so a client cannot
	// widen scope via query params: unscoped admins see all; OPDIV tiers
	// fail-closed to their granted OpDivs' systems; ISSO/ISSM keep the
	// per-system (UserID) path.
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		input.RestrictToOpDivIDs = true
		_, input.OpDivIDs = user.EffectiveOpDivScope()
	default:
		input.UserID = &user.UserID
	}

	if err == nil {
		diffs, err = model.FindScoreDiff(r.Context(), input)
	}

	respond(w, r, diffs, err)
}

//	@Summary		Get per-system questionnaire progress for a data call
//	@Description	Returns, for each FISMA system the caller can see, how many questionnaire functions apply to the system, how many have been genuinely updated in the given data call (answers pre-populated from the previous cycle do not count until touched), and when the most recent update happened. Scoped to the caller's tier: unscoped admins see all systems, OpDiv-scoped admins their OpDivs' systems, and ISSO/ISSM their assigned systems.
//	@Tags		scores
//	@Produce	json
//	@Security	bearerAuth
//	@Param		datacallid		query		int	true	"Data call ID to report progress for"
//	@Param		fismasystemid	query		int	false	"Limit progress to a single FISMA system"
//	@Success	200				{object}	apiResponse[[]model.ScoreProgress]
//	@Failure	400				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/scores/progress [get]
func GetScoresProgress(w http.ResponseWriter, r *http.Request) {
	var (
		progress []*model.ScoreProgress
		err      error
	)

	user := model.UserFromContext(r.Context())
	input := model.FindScoreProgressInput{}

	err = decoder.Decode(&input, r.URL.Query())

	scopeScoreProgressInput(user, &input)

	if err == nil {
		progress, err = model.FindScoreProgress(r.Context(), input)
	}

	respond(w, r, progress, err)
}

// scopeScoreProgressInput applies the caller's tier to the query input. Same
// tier scoping as ListScores, applied AFTER decode so a client cannot widen
// scope via query params: unscoped admins see all; OPDIV tiers fail-closed to
// their granted OpDivs' systems; ISSO/ISSM keep the per-system (UserID) path.
// Extracted so the role matrix is unit-testable without a database.
func scopeScoreProgressInput(user *model.User, input *model.FindScoreProgressInput) {
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		input.RestrictToOpDivIDs = true
		_, input.OpDivIDs = user.EffectiveOpDivScope()
	default:
		input.UserID = &user.UserID
	}
}

//	@Summary	Get aggregated scores
//	@Tags		scores
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	query		int		false	"Filter by FISMA system ID"
//	@Param		datacallid		query		int		false	"Filter by data call ID"
//	@Param		include_pillars	query		bool	false	"Include per-pillar scores"
//	@Success	200				{object}	apiResponse[[]model.ScoreAggregate]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/scores/aggregate [get]
func GetScoresAggregate(w http.ResponseWriter, r *http.Request) {
	var (
		aggregate []*model.ScoreAggregate
		err       error
	)

	user := model.UserFromContext(r.Context())
	findScoresInput := model.FindScoresInput{}

	err = decoder.Decode(&findScoresInput, r.URL.Query())

	// Same tier scoping as ListScores, applied AFTER decode: unscoped admins see
	// all; OPDIV tiers fail-closed to their OpDivs; ISSO/ISSM keep the
	// assigned-systems path.
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		findScoresInput.RestrictToOpDivIDs = true
		_, findScoresInput.OpDivIDs = user.EffectiveOpDivScope()
	default:
		findScoresInput.FismaSystemIDs = user.AssignedFismaSystems
	}

	if err == nil {
		aggregate, err = model.FindScoresAggregate(r.Context(), findScoresInput)
	}

	respond(w, r, aggregate, err)
}
