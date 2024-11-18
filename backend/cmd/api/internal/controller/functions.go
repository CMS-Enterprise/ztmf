package controller

import (
	"fmt"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListQuestionFunctions(w http.ResponseWriter, r *http.Request) {

	input := model.FindFunctionsInput{}

	vars := mux.Vars(r)
	if v, ok := vars["questionid"]; !ok {
		respond(w, r, nil, ErrNotFound)
		return
	} else {
		var questionID int32
		fmt.Sscan(v, &questionID)
		input.QuestionID = &questionID
	}

	functions, err := model.FindFunctions(r.Context(), input)
	respond(w, r, functions, err)
}
