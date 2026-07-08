package controller

import (
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

	if user.IsReadOnlyAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	if !user.IsAdmin() && !user.IsAssignedFismaSystem(score.FismaSystemID) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	// OpDiv write-scope: an admin-tier writer may only score a system in an
	// OpDiv they manage (OWNER/HHS_ADMIN any; OPDIV_ADMIN only their grants).
	// ISSO/ISSM keep the per-system assignment path checked above.
	if user.IsAdmin() {
		if _, err := guardManageFismaSystem(r.Context(), user, score.FismaSystemID); err != nil {
			respond(w, r, nil, err)
			return
		}
	}

	vars := mux.Vars(r)

	if v, ok := vars["scoreid"]; ok {
		fmt.Sscan(v, &scoreID)
		score.ScoreID = scoreID
	}

	score, err = score.Save(r.Context())

	respond(w, r, score, err)
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
		progress, err = model.FindScoreProgress(r.Context(), input)
	}

	respond(w, r, progress, err)
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
