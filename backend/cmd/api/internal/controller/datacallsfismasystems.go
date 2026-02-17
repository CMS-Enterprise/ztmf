package controller

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// SaveDataCallFismaSystem handles the PUT request to mark a FISMA system as having completed a data call
func SaveDataCallFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())

	vars := mux.Vars(r)

	var datacallID, fismasystemID int32

	if v, ok := vars["datacallid"]; ok {
		fmt.Sscan(v, &datacallID)
	}

	if v, ok := vars["fismasystemid"]; ok {
		fmt.Sscan(v, &fismasystemID)
	}

	if !authdUser.IsAdmin() && !authdUser.IsAssignedFismaSystem(fismasystemID) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	df := &model.DataCallFismaSystem{
		Datacallid:    datacallID,
		Fismasystemid: fismasystemID,
	}

	// Call the Save method on this object
	df, err := df.Save(r.Context())

	// Respond with the result
	respond(w, r, df, err)
}

// ListDataCallFismaSystems handles the GET request to list all FISMA systems that have marked a specific data call as complete
func ListDataCallFismaSystems(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var (
		datacallID int32
		v          string
		ok         bool
	)

	if v, ok = vars["datacallid"]; !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	fmt.Sscan(v, &datacallID)

	// Get the list of FISMA systems that have completed this data call
	fismaSystems, err := model.FindDataCallFismaSystems(r.Context(), datacallID)

	// Respond with the result
	respond(w, r, fismaSystems, err)
}

// ListFismaSystemDataCalls handles the GET request to list all data calls that a specific FISMA system has marked as complete
func ListFismaSystemDataCalls(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var (
		fismasystemID int32
		v             string
		ok            bool
	)

	if v, ok = vars["fismasystemid"]; !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	fmt.Sscan(v, &fismasystemID)

	// Get the list of data calls that this FISMA system has completed
	dataCalls, err := model.FindFismaSystemDataCalls(r.Context(), fismasystemID)

	// Respond with the result
	respond(w, r, dataCalls, err)
}
