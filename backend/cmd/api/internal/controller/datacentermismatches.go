package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

// ListDataCenterMismatches returns the wrong-data-center report (ztmf#239):
// active systems whose CFACTS-reported data center environment (from the
// system_enrichment payload) disagrees with the value recorded in ZTMF. This is
// an admin-tier report: unscoped admins see every insights-enabled OpDiv,
// OpDiv-scoped admins see their granted OpDivs (fail-closed), and ISSO/ISSM get
// 403 - an ISSO-facing "report a difference" flow is tracked separately and is
// deliberately not this endpoint.
//
//	@Summary	List systems whose CFACTS data center environment disagrees with ZTMF's
//	@Tags		datacentermismatches
//	@Produce	json
//	@Security	bearerAuth
//	@Success	200	{object}	apiResponse[[]model.DataCenterMismatch]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/datacentermismatches [get]
func ListDataCenterMismatches(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	in := model.FindDataCenterMismatchesInput{}

	// Scope by tier: unscoped admins see all; OpDiv tiers fail-closed to their
	// granted OpDivs' systems; ISSO/ISSM are not the report's audience -> 403.
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		in.RestrictToOpDivIDs = true
		_, in.OpDivIDs = user.EffectiveOpDivScope()
	default:
		respond(w, r, nil, ErrForbidden)
		return
	}

	mismatches, err := model.FindDataCenterMismatches(r.Context(), in)
	respond(w, r, mismatches, err)
}
