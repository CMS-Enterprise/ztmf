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
		input.Userid = &user.Userid
	}

	fismasystems, err := model.FindFismaSystems(r.Context(), input)

	respond(w, fismasystems, err)
}

func GetFismaSystem(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var fismasystemid int32
	fmt.Sscan(mux.Vars(r)["id"], &fismasystemid)
	input := model.FindFismaSystemsInput{
		Fismasystemid: &fismasystemid,
	}

	if !user.IsAdmin() {
		input.Userid = &user.Userid
	}

	fismasystem, err := model.FindFismaSystem(r.Context(), input)
	respond(w, fismasystem, err)
}
