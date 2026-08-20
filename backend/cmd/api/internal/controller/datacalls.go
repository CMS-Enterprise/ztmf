package controller

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/spreadsheet"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

//	@Summary	List all data calls
//	@Tags		datacalls
//	@Produce	json
//	@Security	bearerAuth
//	@Success	200	{object}	apiResponse[[]model.DataCall]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/datacalls [get]
func ListDataCalls(w http.ResponseWriter, r *http.Request) {
	datacalls, err := model.FindDataCalls(r.Context())
	respond(w, r, datacalls, err)
}

//	@Summary	Get a data call by ID
//	@Tags		datacalls
//	@Produce	json
//	@Security	bearerAuth
//	@Param		datacallid	path		int	true	"Data call ID"
//	@Success	200			{object}	apiResponse[model.DataCall]
//	@Failure	404			{object}	apiResponse[any]
//	@Failure	500			{object}	apiResponse[any]
//	@Router		/datacalls/{datacallid} [get]
func GetDataCallByID(w http.ResponseWriter, r *http.Request) {
	var datacallID int32
	vars := mux.Vars(r)
	if v, ok := vars["datacallid"]; !ok {
		respond(w, r, nil, ErrNotFound)
		return
	} else {
		fmt.Sscan(v, &datacallID)
	}

	dc, err := model.FindDataCallByID(r.Context(), datacallID)

	respond(w, r, dc, err)
}

//	@Summary	Export a data call's answers as an xlsx spreadsheet
//	@Tags		datacalls
//	@Produce	application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
//	@Security	bearerAuth
//	@Param		datacallid	path	int		true	"Data call ID"
//	@Param		fsids		query	[]int	false	"FISMA system IDs to filter by"
//	@Success	200	{string}	binary	"xlsx spreadsheet of the data call's answers"
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/datacalls/{datacallid}/export [get]
func GetDatacallExport(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	findAnswersInput := model.FindAnswersInput{}

	// Decode the client query FIRST, then set the path id and the caller's scope,
	// so nothing client-supplied can widen or redirect them. This is the ordering
	// every other list handler uses (see scopeScoreProgressInput); the export was
	// the one place scope was set before decode, which let a query param overwrite
	// it (ztmf-misc#268). The scope fields carry schema:"-" so a decode attempt on
	// them is a 400, but the ordering is the real guard.
	err := decoder.Decode(&findAnswersInput, r.URL.Query())
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// The path is authoritative for the data call; set it after decode so
	// ?DataCallID= cannot redirect the export.
	vars := mux.Vars(r)
	if v, ok := vars["datacallid"]; ok {
		fmt.Sscan(v, &findAnswersInput.DataCallID)
	}

	scopeFindAnswersInput(user, &findAnswersInput)

	answers, err := model.FindAnswers(r.Context(), findAnswersInput)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	file, err := spreadsheet.Excel(answers)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// Fall back to the datacall id when no answers exist so the export still
	// returns a valid, named xlsx (header row only) instead of panicking on
	// answers[0].DataCall.
	filename := fmt.Sprintf("datacall-%d", findAnswersInput.DataCallID)
	if len(answers) > 0 {
		filename = strings.ReplaceAll(answers[0].DataCall, " ", "")
	}
	// Filename is left unquoted because the frontend (FismaTable.saveSystemAnswers)
	// parses the header by splitting on `filename=` and uses the resulting value
	// directly as the anchor's download attribute. Chrome sanitizes filesystem-
	// unsafe characters in that attribute -- including the double-quote -- which
	// turns a quoted filename into _name.xlsx_ and breaks the .xlsx association.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.xlsx", filename))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	// Headers are already on the wire by this point, so a write error cannot
	// be surfaced as a 5xx -- the client will see a truncated download. Log
	// it so server-side observability catches mid-stream disconnects.
	if err := file.Write(w); err != nil {
		log.Printf("GetDatacallExport: error writing xlsx to response (datacallid=%d): %v", findAnswersInput.DataCallID, err)
	}
}

// scopeFindAnswersInput applies the caller's tier to the export query input,
// the same three-way split ListScores and scopeScoreProgressInput use: unscoped
// admins (OWNER, HHS_ADMIN, HHS_READONLY_ADMIN) see every OpDiv; OpDiv tiers
// fail-closed to their granted OpDivs; everyone else is limited to their
// assigned systems. Before this the export used a two-way HasAdminRead() branch
// that dropped both OpDiv tiers into the unscoped path, so an OpDiv admin
// exported every OpDiv's answers (ztmf-misc#267). Extracted so the role matrix
// is unit-testable without a database.
func scopeFindAnswersInput(user *model.User, input *model.FindAnswersInput) {
	if input.ApplyTier(user) {
		input.UserID = user.UserIDPtr()
	}
}

//	@Summary	Create or update a data call
//	@Tags		datacalls
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		datacallid	path	int				false	"Data call ID (for update)"
//	@Param		body		body	model.DataCall	true	"Data call to save"
//	@Success	201	{object}	apiResponse[model.DataCall]
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/datacalls [post]
//	@Router		/datacalls/{datacallid} [put]
func SaveDataCall(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	d := &model.DataCall{}

	err := getJSON(r.Body, d)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	vars := mux.Vars(r)
	if v, ok := vars["datacallid"]; ok {
		fmt.Sscan(v, &d.DataCallID)
	}

	d, err = d.Save(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, d, nil)
}

//	@Summary	Get the latest data call
//	@Tags		datacalls
//	@Produce	json
//	@Security	bearerAuth
//	@Success	200	{object}	apiResponse[model.DataCall]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/datacalls/latest [get]
func GetLatestDataCall(w http.ResponseWriter, r *http.Request) {
	dc, err := model.FindLatestDataCall(r.Context())
	respond(w, r, dc, err)
}
