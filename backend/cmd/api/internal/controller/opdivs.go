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

	if v, ok := mux.Vars(r)["opdiv_id"]; ok {
		fmt.Sscan(v, &o.OpDivID)
	}

	o, err := o.Save(r.Context())
	respond(w, r, o, err)
}
