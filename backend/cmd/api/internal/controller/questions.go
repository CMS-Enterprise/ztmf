package controller

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListQuestions(w http.ResponseWriter, r *http.Request) {
	input := model.FindQuestionInput{}

	vars := mux.Vars(r)
	if v, ok := vars["fismasystemid"]; ok {
		var fismasystemID int32
		fmt.Sscan(v, &fismasystemID)
		input.FismaSystemID = &fismasystemID
	}

	questions, err := model.FindQuestions(r.Context(), input)
	respond(w, r, questions, err)
}
