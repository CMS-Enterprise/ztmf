package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListUserFismaSystems(w http.ResponseWriter, r *http.Request) {
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

	fismasystemids, err := model.FindUserFismaSystemsByUserID(r.Context(), userID)

	respond(w, r, fismasystemids, err)
}

func CreateUserFismaSystem(w http.ResponseWriter, r *http.Request) {
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

	uf := &model.UserFismaSystem{
		UserID: userID,
	}

	err := getJSON(r.Body, uf)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	uf, err = uf.Save(r.Context())
	if err != nil {
		respond(w, r, nil, err)

	}

	respond(w, r, uf, nil)
}

func DeleteUserFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	uf := &model.UserFismaSystem{}

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

	uf.UserID = userID
	fmt.Sscan(fismaSystemID, &uf.FismaSystemID)

	err := uf.Delete(r.Context())

	respond(w, r, "", err)
}
