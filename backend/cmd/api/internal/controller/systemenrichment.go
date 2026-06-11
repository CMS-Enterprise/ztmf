package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// GetSystemEnrichment returns the generic enrichment payload for a FISMA system
// by its fisma_uuid. Access control mirrors the FISMA-system assignment check:
// admins (and read-only admins) may read any system; an ISSO may read only
// systems they are assigned to. A system with no enrichment row yields 404.
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

	if !user.HasAdminRead() {
		canAccess, err := model.UserCanAccessFismaSystemByUUID(r.Context(), user.UserID, fismaUUID)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		if !canAccess {
			respond(w, r, nil, ErrForbidden)
			return
		}
	}

	enrichment, err := model.FindSystemEnrichment(r.Context(), fismaUUID)
	respond(w, r, enrichment, err)
}
