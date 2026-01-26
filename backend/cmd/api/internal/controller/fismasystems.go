package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

func ListFismaSystems(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	input := model.FindFismaSystemsInput{}

	// Decode query parameters (e.g., ?decommissioned=true)
	err := decoder.Decode(&input, r.URL.Query())
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	if !user.IsAdmin() {
		input.UserID = &user.UserID
	}

	fismasystems, err := model.FindFismaSystems(r.Context(), input)

	respond(w, r, fismasystems, err)
}

func GetFismaSystem(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
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
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	f := &model.FismaSystem{}

	err := getJSON(r.Body, f)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	vars := mux.Vars(r)
	if v, ok := vars["fismasystemid"]; ok {
		fmt.Sscan(v, &f.FismaSystemID)
	}

	f, err = f.Save(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, f, nil)
}

// DecommissionRequest contains optional parameters for decommissioning
type DecommissionRequest struct {
	DecommissionedDate *string `json:"decommissioned_date,omitempty"`
	Notes              *string `json:"notes,omitempty"`
}

// DeleteFismaSystem handles the decommissioning of a fismasystem
func DeleteFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	fismaSystemIDStr, ok := vars["fismasystemid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	var fismaSystemID int32
	fmt.Sscan(fismaSystemIDStr, &fismaSystemID)

	// Parse optional request body
	var req DecommissionRequest
	if r.ContentLength > 0 {
		if err := getJSON(r.Body, &req); err != nil {
			log.Println(err)
			respond(w, r, nil, ErrMalformed)
			return
		}
	}

	// Build decommission input
	input := model.DecommissionInput{
		FismaSystemID: fismaSystemID,
		UserID:        authdUser.UserID,
		Notes:         req.Notes,
	}

	// Parse date if provided
	if req.DecommissionedDate != nil {
		parsedDate, err := parseRFC3339(*req.DecommissionedDate)
		if err != nil {
			log.Println(err)
			respond(w, r, nil, ErrMalformed)
			return
		}
		input.DecommissionedDate = &parsedDate
	}

	err := model.DeleteFismaSystem(r.Context(), input)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respond(w, r, nil, nil)
}
