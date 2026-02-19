package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

func ListCfactsSystems(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	if !user.HasAdminRead() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	input := model.FindCfactsSystemsInput{}
	err := decoder.Decode(&input, r.URL.Query())
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	systems, err := model.FindCfactsSystems(r.Context(), input)

	respond(w, r, systems, err)
}

func GetCfactsSystem(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	if !user.HasAdminRead() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	fismaUUID, ok := vars["fisma_uuid"]
	if !ok || fismaUUID == "" {
		respond(w, r, nil, ErrNotFound)
		return
	}

	system, err := model.FindCfactsSystem(r.Context(), fismaUUID)
	respond(w, r, system, err)
}
