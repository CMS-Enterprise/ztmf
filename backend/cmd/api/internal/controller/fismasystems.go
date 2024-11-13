package controller

import (
	"fmt"
	"log"
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

	respond(w, r, fismasystems, err)
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
		respond(w, r, nil, ErrForbidden)
		return
	}

	fismasystem, err := model.FindFismaSystem(r.Context(), input)
	respond(w, r, fismasystem, err)
}

func SaveFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := auth.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	fismasystem := &model.FismaSystem{}

	err := getJSON(r.Body, fismasystem)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	vars := mux.Vars(r)
	if v, ok := vars["fismasystemid"]; ok {
		fmt.Sscan(v, &fismasystem.FismaSystemID)
	}

	if fismasystem.FismaSystemID == 0 {
		fismasystem, err = model.CreateFismaSystem(r.Context(), *fismasystem)
	} else {
		fismasystem, err = model.UpdateFismaSystem(r.Context(), *fismasystem)
	}

	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respond(w, r, fismasystem, nil)
}
