package controller

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListFismaSystems(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	input := model.FindFismaSystemsInput{}

	if !user.IsAdmin() {
		input.UserID = &user.UserID
	}

	fismasystems, err := model.FindFismaSystems(r.Context(), input)

	respond(w, fismasystems, err)
}

func GetFismaSystem(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	vars := mux.Vars(r)
	input := model.FindFismaSystemsInput{}

	if v, ok := vars["fismasystemid"]; ok {
		var fismasystemID int32
		fmt.Sscan(v, &fismasystemID)
		input.FismaSystemID = &fismasystemID
	}

	if !user.IsAdmin() && !user.IsAssignedFismaSystem(*input.FismaSystemID) {
		respond(w, nil, &ForbiddenError{})
		return
	}

	fismasystem, err := model.FindFismaSystem(r.Context(), input)
	respond(w, fismasystem, err)
}
