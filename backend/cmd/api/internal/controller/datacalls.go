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
//	@Description	The name (datacall) is a free-text title, unique ignoring case. Create requires datacall, deadline and fiscal_year; cadence defaults to "annual". Update is partial: any field left out keeps its stored value. A path id that is zero or out of range is a 404.
//	@Tags		datacalls
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		datacallid	path	int				false	"Data call ID (for update)"
//	@Param		body		body	model.DataCall	true	"Data call to save"
//	@Success	201	{object}	apiResponse[model.DataCall]
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
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

	// The route decides create vs update, never the body: getJSON decodes a
	// "datacallid" field, which would otherwise turn a POST into an update of
	// that row.
	d.DataCallID = 0
	// A path id that is present but unusable (0, or past int32) is a 404,
	// never a fallthrough to the create branch: the route said update.
	// pathInt32 also reads "0012" as 12, where fmt.Sscan read it as octal 10.
	if v, ok := mux.Vars(r)["datacallid"]; ok {
		id, valid := pathInt32(v)
		if !valid {
			respond(w, r, nil, ErrNotFound)
			return
		}
		d.DataCallID = id
	}

	d, err = d.Save(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, d, nil)
}

type repinDataCallInput struct {
	QuestionnaireVersionID int32 `json:"questionnaire_versionid"`
}

// RepinDataCall moves an open data call to another published questionnaire
// version, translating its answers (ztmf-misc#397). It changes what every
// OpDiv answers this cycle, so it takes the catalog's CanWriteHHSWide gate,
// checked before the body is read.
//
//	@Summary		Pin a data call to another questionnaire version
//	@Description	Answers move by function lineage and option score. Until anyone has answered this cycle, answers with no counterpart are dropped; after that, every answer must carry over or the re-pin is a 409 VERSION_NOT_TRANSLATABLE. Translated answers that have history gain a translate revision, which undo refuses. Refused for a draft or archived version and after the deadline.
//	@Tags			datacalls
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			datacallid	path		int					true	"Data call ID"
//	@Param			body		body		repinDataCallInput	true	"Target version"
//	@Success		200			{object}	apiResponse[model.DataCallRepin]
//	@Failure		400			{object}	apiResponse[any]
//	@Failure		403			{object}	apiResponse[any]
//	@Failure		404			{object}	apiResponse[any]
//	@Failure		409			{object}	apiResponse[any]
//	@Failure		500			{object}	apiResponse[any]
//	@Router			/datacalls/{datacallid}/questionnaire-version [put]
func RepinDataCall(w http.ResponseWriter, r *http.Request) {
	if !model.UserFromContext(r.Context()).CanWriteHHSWide() {
		respond(w, r, nil, ErrForbidden)
		return
	}
	id, ok := pathInt32(mux.Vars(r)["datacallid"])
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}
	input := repinDataCallInput{}
	if err := getJSON(r.Body, &input); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}
	res, err := model.RepinDataCall(r.Context(), id, input.QuestionnaireVersionID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	respondOK(w, res)
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
