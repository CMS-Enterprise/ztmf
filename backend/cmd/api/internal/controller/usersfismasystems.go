package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

func ListUserFismaSystems(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.HasAdminRead() {
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

	// OpDiv write-scope: the acting admin may only assign a system they manage
	// to a user they manage. OWNER/HHS_ADMIN pass both; an OPDIV_ADMIN must hold
	// the system's OpDiv and share an OpDiv with the target user.
	if _, gerr := guardManageFismaSystem(r.Context(), authdUser, uf.FismaSystemID); gerr != nil {
		respond(w, r, nil, gerr)
		return
	}
	target, terr := model.FindUserByID(r.Context(), userID)
	if terr != nil {
		respond(w, r, nil, terr)
		return
	}
	if !authdUser.CanManageUser(target) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	uf, err = uf.Save(r.Context())
	if err != nil {
		respond(w, r, nil, err)
		return
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

	// Same OpDiv write-scope as assignment: manage both the system and the user.
	if _, gerr := guardManageFismaSystem(r.Context(), authdUser, uf.FismaSystemID); gerr != nil {
		respond(w, r, nil, gerr)
		return
	}
	target, terr := model.FindUserByID(r.Context(), userID)
	if terr != nil {
		respond(w, r, nil, terr)
		return
	}
	if !authdUser.CanManageUser(target) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	err := uf.Delete(r.Context())

	respond(w, r, "", err)
}
