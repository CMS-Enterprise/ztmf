package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

func ListCfactsSystems(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	input := model.FindCfactsSystemsInput{}
	err := decoder.Decode(&input, r.URL.Query())
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	if !user.HasAdminRead() {
		input.UserID = &user.UserID
	}

	systems, err := model.FindCfactsSystems(r.Context(), input)

	respond(w, r, systems, err)
}

func GetCfactsSystem(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	vars := mux.Vars(r)
	fismaUUID, ok := vars["fisma_uuid"]
	if !ok || fismaUUID == "" {
		respond(w, r, nil, ErrNotFound)
		return
	}

	if !user.HasAdminRead() {
		canAccess, err := model.UserCanAccessCfactsSystem(r.Context(), user.UserID, fismaUUID)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		if !canAccess {
			respond(w, r, nil, ErrForbidden)
			return
		}
	}

	system, err := model.FindCfactsSystem(r.Context(), fismaUUID)
	respond(w, r, system, err)
}
