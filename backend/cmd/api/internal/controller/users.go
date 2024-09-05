package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListUsers(w http.ResponseWriter, r *http.Request) {
	// TODO: replace the repititious admin checks with ACL
	authdUser := auth.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, nil, &ForbiddenError{})
		return
	}

	users, err := model.FindUsers(r.Context())

	respond(w, users, err)
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

func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	respond(w, user, nil)
}

// SaveUser is for admin management
func SaveUser(w http.ResponseWriter, r *http.Request) {
	authdUser := auth.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, nil, &ForbiddenError{})
		return
	}

	user := &model.User{}

	err := getJSON(r.Body, user)
	if err != nil {
		log.Println(err)
		respond(w, nil, err)
		return
	}

	err = validateEmail(user.Email)
	if err != nil {
		respond(w, nil, err)
		return
	}

	err = validateRole(user.Role)
	if err != nil {
		respond(w, nil, err)
		return
	}

	vars := mux.Vars(r)
	if v, ok := vars["userid"]; ok {
		user.UserID = v
	}

	if user.UserID != "" {
		user, err = model.UpdateUser(r.Context(), *user)
	} else {
		user, err = model.CreateUser(r.Context(), *user)
	}

	if err != nil {
		log.Println(err)
		// TODO: wrap all such db errors to return generic 500 to client but still log it
		respond(w, nil, err)
		return
	}

	respond(w, user, nil)
}
