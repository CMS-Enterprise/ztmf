package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

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
		functionID := pathInt32(v)
		input.FunctionID = &functionID
	}

	functionoptions, err := model.FindFunctionOptions(r.Context(), input)
	respond(w, r, functionoptions, err)
}

//	@Summary	Create or update an answer option
//	@Tags		functions
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		functionid			path		int						false	"Function ID (create)"
//	@Param		functionoptionid	path		int						false	"Function Option ID (update)"
//	@Param		body				body		model.FunctionOption	true	"Option to create or update"
//	@Success	201					{object}	apiResponse[model.FunctionOption]
//	@Success	204					"No Content"
//	@Failure	400					{object}	apiResponse[any]
//	@Failure	403					{object}	apiResponse[any]
//	@Failure	404					{object}	apiResponse[any]
//	@Failure	500					{object}	apiResponse[any]
//	@Router		/functions/{functionid}/options [post]
//	@Router		/functionoptions/{functionoptionid} [put]
func SaveFunctionOption(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	// HHS-wide catalog, same reasoning as SaveQuestion (ztmf-misc#398). The read
	// sibling above carries no gate at all, so this one is stated explicitly
	// rather than inherited.
	if !user.CanWriteHHSWide() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	fo := &model.FunctionOption{}

	if err := getJSON(r.Body, fo); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	// Both ids are re-pinned from the route after decoding, never taken from the
	// body. getJSON decodes any field present on the struct, so without this a
	// POST carrying "functionoptionid" would be rerouted into an update of
	// somebody else's option - the path ignored entirely - and a body
	// "functionid" would retarget the write to a function the caller never
	// named. The route decides create vs update; the body only carries content.
	pathOptionID, isUpdate := vars["functionoptionid"]
	fo.FunctionOptionID = 0

	if isUpdate {
		fo.FunctionOptionID = pathInt32(pathOptionID)
		// PUT carries no functionid in the path, so the stored row is the only
		// trustworthy source for which function this option belongs to.
		stored, err := model.FindFunctionOptionByID(r.Context(), fo.FunctionOptionID)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		fo.FunctionID = stored.FunctionID
	} else {
		fo.FunctionID = 0
		if v, ok := vars["functionid"]; ok {
			fo.FunctionID = pathInt32(v)
		}
	}

	saved, err := fo.Save(r.Context())
	respond(w, r, saved, err)
}
