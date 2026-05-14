package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
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
