package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

//	@Summary	List all functions
//	@Tags		functions
//	@Produce	json
//	@Security	bearerAuth
//	@Success	200	{object}	apiResponse[[]model.Function]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/functions [get]
func ListFunctions(w http.ResponseWriter, r *http.Request) {

	var (
		functions []*model.Function
		err       error
	)

	findFunctionsInput := model.FindFunctionsInput{}
	err = decoder.Decode(&findFunctionsInput, r.URL.Query())
	if err == nil {
		functions, err = model.FindFunctions(r.Context(), findFunctionsInput)
	}
	respond(w, r, functions, err)
}

//	@Summary	Get a function by ID
//	@Tags		functions
//	@Produce	json
//	@Security	bearerAuth
//	@Param		functionid	path		int	true	"Function ID"
//	@Success	200			{object}	apiResponse[model.Function]
//	@Failure	404			{object}	apiResponse[any]
//	@Failure	500			{object}	apiResponse[any]
//	@Router		/functions/{functionid} [get]
func GetFunctionByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID, ok := vars["functionid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}
	var functionID int32
	fmt.Sscan(ID, &functionID)

	f, err := model.FindFunctionByID(r.Context(), functionID)

	respond(w, r, f, err)
}

//	@Summary	Create or update a function
//	@Tags		functions
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		functionid	path		int				true	"Function ID"
//	@Param		body		body		model.Function	true	"Function to create or update"
//	@Success	201			{object}	apiResponse[model.Function]
//	@Failure	400			{object}	apiResponse[any]
//	@Failure	403			{object}	apiResponse[any]
//	@Failure	500			{object}	apiResponse[any]
//	@Router		/functions [post]
//	@Router		/functions/{functionid} [put]
func SaveFunction(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	if !user.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	f := &model.Function{}

	err := getJSON(r.Body, f)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	vars := mux.Vars(r)
	if v, ok := vars["functionid"]; ok {
		fmt.Sscan(v, &f.FunctionID)
	}

	f, err = f.Save(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, f, nil)

}
