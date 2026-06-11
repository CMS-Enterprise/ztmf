package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// ListScores godoc
//
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

// SaveScore godoc
//
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

// GetScoresAggregate godoc
//
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
