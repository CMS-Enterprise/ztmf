package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

// ListDataCenterEnvironments returns the datacenterenvironment reference list.
// Open to any authenticated user because it carries no sensitive data and the
// frontend needs it to build the system-environment dropdown from the backend
// (ztmf#392) instead of a hardcoded list. Pass selectable_only=true to get only
// the values offered for new/edited systems.
//
//	@Summary	List datacenterenvironment mappings
//	@Tags		datacenterenvironments
//	@Produce	json
//	@Security	bearerAuth
//	@Param		selectable_only	query		bool	false	"Only environments offered in the system dropdown"
//	@Success	200				{object}	apiResponse[[]model.DataCenterEnvironment]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/datacenterenvironments [get]
func ListDataCenterEnvironments(w http.ResponseWriter, r *http.Request) {
	input := model.FindDataCenterEnvironmentsInput{}

	if err := decoder.Decode(&input, r.URL.Query()); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	envs, err := model.FindDataCenterEnvironments(r.Context(), input)
	respond(w, r, envs, err)
}
