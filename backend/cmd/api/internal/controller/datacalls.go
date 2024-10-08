package controller

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/spreadsheet"
	"github.com/gorilla/mux"
)

func ListDataCalls(w http.ResponseWriter, r *http.Request) {
	datacalls, err := model.FindDataCalls(r.Context())
	respond(w, r, datacalls, err)
}

func GetDatacallExport(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	input := model.FindAnswersInput{}

	if !user.IsAdmin() {
		input.UserID = &user.UserID
	}

	vars := mux.Vars(r)
	if v, ok := vars["datacallid"]; ok {
		fmt.Sscan(v, &input.DataCallID)
	}

	qVars := r.URL.Query()
	if qVars.Has("fsids") {
		for _, v := range qVars["fsids"] {
			var fismaSystemID int32
			fmt.Sscan(v, &fismaSystemID)
			if !user.IsAdmin() && !user.IsAssignedFismaSystem(fismaSystemID) {
				respond(w, r, nil, ErrForbidden)
				return
			}
			input.FismaSystemIDs = append(input.FismaSystemIDs, &fismaSystemID)
		}
	}

	answers, err := model.FindAnswers(r.Context(), input)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	file, err := spreadsheet.Excel(answers)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+answers[0].DataCall+".xslx")
	w.Header().Set("Content-Type", "application/vnd.ms-excel")
	file.Write(w)
}
