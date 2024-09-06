package controller

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListFunctionOptions(w http.ResponseWriter, r *http.Request) {
	input := model.FindFunctionOptionsInput{}

	vars := mux.Vars(r)
	if v, ok := vars["functionid"]; ok {
		var functionID int32
		fmt.Sscan(v, &functionID)
		input.FunctionID = &functionID
	}

	functionoptions, err := model.FindFunctionOptions(r.Context(), input)
	respond(w, r, functionoptions, err)
}
