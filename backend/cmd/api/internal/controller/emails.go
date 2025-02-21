package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
)

// SendEmails only responds to a PUT request so as to update a single item table
// see model/emails
func SaveEmail(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	if !user.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	m := &model.MassEmail{}

	err := getJSON(r.Body, m)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	m, err = m.Save(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, m, nil)
}
