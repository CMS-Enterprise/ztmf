package controller

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// ListFunctionOptions godoc
//
//	@Summary	List options for a function
//	@Tags		functions
//	@Produce	json
//	@Security	bearerAuth
//	@Param		functionid	path		int	true	"Function ID"
//	@Success	200			{object}	apiResponse[[]model.FunctionOption]
//	@Failure	500			{object}	apiResponse[any]
//	@Router		/functions/{functionid}/options [get]
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
