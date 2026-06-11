package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/mail"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

// SaveMassEmail only responds to a PUT request so as to update a single item table
// see model/massemails
//	@Summary	Send a mass email to a recipient group
//	@Tags		massemails
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		body	body		model.MassEmail	true	"Mass email subject, body, and recipient group"
//	@Success	201	{object}	apiResponse[[]string]
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/massemails [post]
func SaveMassEmail(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	// Mass email is a write action that targets recipients across every OpDiv
	// with no per-OpDiv recipient scoping, so it is restricted to the unscoped
	// WRITE admins (OWNER / HHS_ADMIN): IsAdmin excludes the read-only tiers and
	// HasUnscopedRead excludes OPDIV_ADMIN. An OPDIV_ADMIN must not be able to
	// blast users outside their OpDiv; read-only admins must not send at all.
	if !user.IsAdmin() || !user.HasUnscopedRead() {
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

	recipients, err := m.Recipients(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	go mail.Send(m.Subject, m.Body, recipients)

	respond(w, r, recipients, nil)
}
