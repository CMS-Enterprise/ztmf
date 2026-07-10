package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

//	@Summary	List per-system per-question insights
//	@Tags		insights
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	query		int	false	"Filter by FISMA system ID"
//	@Param		questionid		query		int	false	"Filter by question ID"
//	@Success	200				{object}	apiResponse[[]model.SystemInsight]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/insights [get]
func ListSystemInsights(w http.ResponseWriter, r *http.Request) {
	var (
		insights []*model.SystemInsight
		err      error
	)
	user := model.UserFromContext(r.Context())
	in := model.FindSystemInsightsInput{}

	err = decoder.Decode(&in, r.URL.Query())

	// Scope by tier AFTER decode so a client cannot widen scope via query
	// params: unscoped admins see all; OpDiv tiers fail-closed to their granted
	// OpDivs' systems; ISSO/ISSM keep the per-system (UserID) path.
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		in.RestrictToOpDivIDs = true
		_, in.OpDivIDs = user.EffectiveOpDivScope()
	default:
		in.UserID = &user.UserID
	}

	if err == nil {
		insights, err = model.FindSystemInsights(r.Context(), in)
	}

	respond(w, r, insights, err)
}
