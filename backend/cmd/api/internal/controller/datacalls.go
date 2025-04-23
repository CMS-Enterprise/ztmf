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

func ListDataCalls(w http.ResponseWriter, r *http.Request) {
	datacalls, err := model.FindDataCalls(r.Context())
	respond(w, r, datacalls, err)
}

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

func GetDatacallExport(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	findAnswersInput := model.FindAnswersInput{}

	if !user.IsAdmin() {
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

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.xlsx", strings.ReplaceAll(answers[0].DataCall, " ", "")))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	file.Write(w)
}

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

func GetLatestDataCall(w http.ResponseWriter, r *http.Request) {
	dc, err := model.FindLatestDataCall(r.Context())
	respond(w, r, dc, err)
}
