package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListUsers(w http.ResponseWriter, r *http.Request) {
	// TODO: replace the repititious admin checks with ACL
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	users, err := model.FindUsers(r.Context())

	respond(w, r, users, err)
}

func GetUserByID(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	ID, ok := vars["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	user, err := model.FindUserByID(r.Context(), ID)

	respond(w, r, user, err)
}

func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	respond(w, r, user, nil)
}

// SaveUser is for admin management
func SaveUser(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	user := &model.User{}

	err := getJSON(r.Body, user)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	vars := mux.Vars(r)
	if v, ok := vars["userid"]; ok {
		user.UserID = v
	}

	user, err = user.Save(r.Context())

	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respond(w, r, user, nil)
}

// DeleteUser handles the deletion of a user
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	userID, ok := vars["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	err := model.DeleteUser(r.Context(), userID)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respond(w, r, nil, nil)
}
