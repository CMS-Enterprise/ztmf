package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func GetUserById(w http.ResponseWriter, r *http.Request) {
	authdUser := auth.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, nil, &ForbiddenError{})
		return
	}

	vars := mux.Vars(r)
	ID, ok := vars["userid"]
	if !ok {
		respond(w, nil, &InvalidInputError{"id", nil})
		return
	}

	user, err := model.FindUserByID(r.Context(), ID)

	respond(w, user, err)
}

func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	respond(w, user, nil)
}
