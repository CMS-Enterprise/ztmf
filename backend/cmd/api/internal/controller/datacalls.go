package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
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

	if !user.HasAdminRead() {
		findAnswersInput.UserID = &user.UserID
	}

	vars := mux.Vars(r)
	if v, ok := vars["datacallid"]; ok {
		fmt.Sscan(v, &findAnswersInput.DataCallID)
	}

	err := decoder.Decode(&findAnswersInput, r.URL.Query())
	if err != nil {
		respond(w, r, nil, err)
		return
	}

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

//	@Summary	Export a data call's questionnaire time-spent analytics as an xlsx spreadsheet
//	@Tags		datacalls
//	@Produce	application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
//	@Security	bearerAuth
//	@Param		datacallid	path	int		true	"Data call ID"
//	@Param		fsids		query	[]int	false	"FISMA system IDs to filter by"
//	@Success	200	{string}	binary	"xlsx spreadsheet of the data call's time-spent analytics"
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/datacalls/{datacallid}/export/timespent [get]
func GetDatacallTimeSpentExport(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	var datacallID int32
	vars := mux.Vars(r)
	if v, ok := vars["datacallid"]; ok {
		fmt.Sscan(v, &datacallID)
	}

	// fsids is the optional CSV system filter, mirroring the answers export's
	// query param. Decoded into a throwaway input so a client cannot inject
	// scope via query params - tier scope is applied below, not from the query.
	var q struct {
		FismaSystemIDs []int32 `schema:"fsids"`
	}
	if err := decoder.Decode(&q, r.URL.Query()); err != nil {
		respond(w, r, nil, err)
		return
	}

	input := model.FindTimeSpentInput{DataCallID: &datacallID}

	// Same tier scoping as the other score reads, applied server-side: unscoped
	// admins see all; OPDIV tiers fail-closed to their granted OpDivs' systems;
	// ISSO/ISSM keep the per-system (UserID) path.
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		input.RestrictToOpDivIDs = true
		_, input.OpDivIDs = user.EffectiveOpDivScope()
	default:
		input.UserID = &user.UserID
	}

	timeSpent, err := model.FindTimeSpent(r.Context(), input)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// Resolve display acronym/name for every system the caller can see. Scoped
	// the same way as the analytics so a caller never resolves (or lists) a
	// system they cannot see. Doubles as the accessibility gate for the
	// requested-system list below. The spreadsheet package stays database-free
	// and takes the lookup.
	sysInfo, err := resolveSystemInfo(r.Context(), user)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// Emit a row for EVERY requested, visible system - not only those with
	// recorded activity - so a system nobody has opened yet still appears (with
	// zeroed metrics and a "no activity" status), and the export is never an
	// empty, header-only file.
	timeSpent = selectExportSystems(timeSpent, sysInfo, q.FismaSystemIDs)

	// Resolve question id -> text for the Per Question sheet's display column.
	// Questions are not system-scoped (the full catalog is readable), so no
	// per-caller filtering is needed here.
	questions, err := model.FindQuestions(r.Context())
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	questionText := make(map[int32]string, len(questions))
	for _, q := range questions {
		questionText[q.QuestionID] = q.Question
	}

	// Resolve the data call's human-readable name so the workbook is
	// self-identifying (shown on each sheet) and the filename is recognizable.
	// Best-effort: on a lookup miss the numeric id still labels both. Any real
	// DB error would already have surfaced from FindTimeSpent above.
	dataCallLabel := fmt.Sprintf("#%d", datacallID)
	filename := fmt.Sprintf("datacall-%d-timespent", datacallID)
	if dc, err := model.FindDataCallByID(r.Context(), datacallID); err == nil && dc != nil && dc.DataCall != "" {
		dataCallLabel = dc.DataCall
		filename = strings.ReplaceAll(dc.DataCall, " ", "") + "-timespent"
	}

	file, err := spreadsheet.TimeSpentExcel(timeSpent, sysInfo, questionText, dataCallLabel)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// Filename left unquoted to match GetDatacallExport (the frontend parses the
	// header by splitting on `filename=`; a quoted value breaks the download).
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.xlsx", filename))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := file.Write(w); err != nil {
		log.Printf("GetDatacallTimeSpentExport: error writing xlsx to response (datacallid=%d): %v", datacallID, err)
	}
}

// resolveSystemInfo loads the acronym/name for every system the caller can see,
// keyed by id, for the time-spent export's display columns. Scoped to the
// caller's tier so it never surfaces a system the analytics query would exclude.
func resolveSystemInfo(ctx context.Context, user *model.User) (map[int32]spreadsheet.SystemInfo, error) {
	in := model.FindFismaSystemsInput{}
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		in.RestrictToOpDivIDs = true
		_, in.OpDivIDs = user.EffectiveOpDivScope()
	default:
		in.UserID = &user.UserID
	}

	systems, err := model.FindFismaSystems(ctx, in)
	if err != nil {
		return nil, err
	}

	out := make(map[int32]spreadsheet.SystemInfo, len(systems))
	for _, s := range systems {
		out[s.FismaSystemID] = spreadsheet.SystemInfo{Acronym: s.FismaAcronym, Name: s.FismaName}
	}
	return out, nil
}

// selectExportSystems returns the systems the time-spent export should render:
// one entry per requested, visible system, reusing the analytics result where a
// system had activity and filling a zeroed TimeSpent where it did not - so a
// not-yet-opened system still appears rather than dropping out of the file.
//
// requested is the fsids filter (empty means "every visible system"); visible is
// the caller's tier-scoped set, which also gates access - an out-of-scope id is
// silently dropped so the export never names a system the caller cannot see.
// Order follows the requested list, or ascending id when unfiltered, for a
// stable export.
func selectExportSystems(results []*model.TimeSpent, visible map[int32]spreadsheet.SystemInfo, requested []int32) []*model.TimeSpent {
	byID := make(map[int32]*model.TimeSpent, len(results))
	for _, ts := range results {
		byID[ts.FismaSystemID] = ts
	}

	var targets []int32
	if len(requested) > 0 {
		for _, id := range requested {
			if _, ok := visible[id]; ok {
				targets = append(targets, id)
			}
		}
	} else {
		for id := range visible {
			targets = append(targets, id)
		}
		sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })
	}

	out := make([]*model.TimeSpent, 0, len(targets))
	for _, id := range targets {
		if ts, ok := byID[id]; ok {
			out = append(out, ts)
		} else {
			out = append(out, &model.TimeSpent{FismaSystemID: id})
		}
	}
	return out
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
