package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

// ListSystemAttributes returns the systemattributes reference list: the
// canonical allowed values per FISMA system metadata field (ztmf#395). Open to
// any authenticated user because it carries no sensitive data and the frontend
// needs it to build the metadata selects from the backend instead of hardcoded
// lists, the same pattern as ListDataCenterEnvironments (ztmf#392). Pass
// field=<name> to scope to one field and selectable_only=true to get only the
// values offered for new/edited systems (hides legacy-alias and help rows).
//
//	@Summary	List system metadata attribute vocabulary
//	@Tags		systemattributes
//	@Produce	json
//	@Security	bearerAuth
//	@Param		field			query		string	false	"Restrict to a single attribute (e.g. system_type)"
//	@Param		selectable_only	query		bool	false	"Only values offered in the add/edit selects"
//	@Success	200				{object}	apiResponse[[]model.SystemAttribute]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/systemattributes [get]
func ListSystemAttributes(w http.ResponseWriter, r *http.Request) {
	input := model.FindSystemAttributesInput{}

	if err := decoder.Decode(&input, r.URL.Query()); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	attrs, err := model.FindSystemAttributes(r.Context(), input)
	respond(w, r, attrs, err)
}
