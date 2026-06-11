package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// ListOpDivs returns the OpDiv reference list. Open to any authenticated user
// because the list contains no sensitive data and the frontend needs it for
// OpDiv selectors (admin user-create, system-create, importer validation).
//
//	@Summary	List all OpDivs
//	@Tags		opdivs
//	@Produce	json
//	@Security	bearerAuth
//	@Success	200	{object}	apiResponse[[]model.OpDiv]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/opdivs [get]
func ListOpDivs(w http.ResponseWriter, r *http.Request) {
	input := model.FindOpDivsInput{}

	if err := decoder.Decode(&input, r.URL.Query()); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	opdivs, err := model.FindOpDivs(r.Context(), input)
	respond(w, r, opdivs, err)
}

// SaveOpDiv creates (POST) or updates (PUT) an OpDiv. Restricted to OWNER: the
// OpDiv list is the tenant boundary itself, so only the unscoped platform tier
// may add or change one. A PUT with active=false deactivates an OpDiv. This is
// the runtime path for onboarding a new OpDiv without a code deploy.
//
//	@Summary	Create or update an OpDiv
//	@Tags		opdivs
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		opdiv_id	path		int			true	"OpDiv ID"
//	@Param		body		body		model.OpDiv	true	"OpDiv to create or update"
//	@Success	201			{object}	apiResponse[model.OpDiv]
//	@Failure	400			{object}	apiResponse[any]
//	@Failure	403			{object}	apiResponse[any]
//	@Failure	500			{object}	apiResponse[any]
//	@Router		/opdivs [post]
//	@Router		/opdivs/{opdiv_id} [put]
func SaveOpDiv(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsOwner() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	o := &model.OpDiv{}
	if err := getJSON(r.Body, o); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	// Trust only the path-derived id. POST has no {opdiv_id} path var, so any
	// opdiv_id in the request body is ignored and the operation is always a
	// create - a body-supplied id must not turn a POST into an update of an
	// existing OpDiv.
	o.OpDivID = 0
	if v, ok := mux.Vars(r)["opdiv_id"]; ok {
		fmt.Sscan(v, &o.OpDivID)
	}

	o, err := o.Save(r.Context())
	respond(w, r, o, err)
}
