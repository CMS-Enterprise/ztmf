package controller

import (
	"context"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func GetUser(ctx context.Context, userid string) (*model.User, error) {
	user := auth.UserFromContext(ctx)

	if !user.IsAdmin() && user.UserID != userid {
		return nil, nil
	}
	return model.FindUserByID(ctx, userid)
}

func GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	authdUser := auth.UserFromContext(r.Context())
	vars := mux.Vars(r)
	email, ok := vars["email"]
	if !ok {
		respond(w, nil, &InvalidInputError{"email", nil})
		return
	}

	if !authdUser.IsAdmin() && email != authdUser.Email {
		respond(w, nil, &ForbiddenError{})
		return
	}

	user, err := model.FindUserByEmail(r.Context(), email)

	respond(w, user, err)
}

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
