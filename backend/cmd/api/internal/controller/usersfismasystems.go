package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListUserFismaSystems(w http.ResponseWriter, r *http.Request) {
	authdUser := auth.UserFromContext(r.Context())
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

	fismasystemids, err := model.FindUserFismaSystemsByUserID(r.Context(), userID)

	respond(w, r, fismasystemids, err)
}

func CreateUserFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := auth.UserFromContext(r.Context())
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

	userFismaSystem := &model.UserFismaSystem{
		UserID: userID,
	}

	err := getJSON(r.Body, userFismaSystem)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	err = model.AddUserFismaSystem(r.Context(), *userFismaSystem)
	if err != nil {
		userFismaSystem = nil
	}
	respond(w, r, userFismaSystem, err)
}

func DeleteUserFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := auth.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	userFismaSystem := model.UserFismaSystem{}

	vars := mux.Vars(r)
	userID, ok := vars["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	fismaSystemID, ok := vars["fismasystemid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	userFismaSystem.UserID = userID
	fmt.Sscan(fismaSystemID, &userFismaSystem.FismaSystemID)

	err := model.DeleteUserFismaSystem(r.Context(), userFismaSystem)

	respond(w, r, "", err)
}
