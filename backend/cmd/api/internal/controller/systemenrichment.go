package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// GetSystemEnrichment returns the generic enrichment payload for a FISMA system
// by its fisma_uuid. Access is OpDiv-scoped (fetch-then-gate): unscoped admin
// tiers read any system, OpDiv-scoped admins read systems in their granted
// OpDivs, and an ISSO reads only systems they are assigned to. Enrichment is
// additionally gated on the owning OpDiv having insights_enabled = TRUE (handled
// in the model), so a system in a non-enabled OpDiv yields 404 - the same
// response as a system with no enrichment row. A caller who lacks access to an
// existing, insights-enabled system still gets 403 (consistent with the rest of
// the API's authz); this hides the gate and missing systems, not the existence
// of systems the caller simply cannot read.
//	@Summary	Get enrichment payload for a FISMA system
//	@Tags		systemenrichment
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fisma_uuid	path		string	true	"FISMA system UUID"
//	@Success	200	{object}	apiResponse[model.SystemEnrichment]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/systemenrichment/{fisma_uuid} [get]
func GetSystemEnrichment(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	vars := mux.Vars(r)
	fismaUUID, ok := vars["fisma_uuid"]
	if !ok || fismaUUID == "" {
		respond(w, r, nil, ErrNotFound)
		return
	}

	// Resolve the system first so access is evaluated against its OpDiv. A miss
	// stays 404 (ErrNoData) so we never leak which systems exist via a 403.
	system, err := model.FindFismaSystemByUUID(r.Context(), fismaUUID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	if !user.CanAccessFismaSystem(system.OpDivID, system.FismaSystemID) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	enrichment, err := model.FindSystemEnrichment(r.Context(), fismaUUID)
	respond(w, r, enrichment, err)
}
