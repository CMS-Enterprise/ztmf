package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
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

func SaveFunction(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !user.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	f := &model.Function{}

	vars := mux.Vars(r)
	if v, ok := vars["questionid"]; !ok {
		respond(w, r, nil, ErrNotFound)
		return
	} else {
		fmt.Sscan(v, &f.QuestionID)
	}

	err := getJSON(r.Body, f)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	if v, ok := vars["functionid"]; ok {
		var functionID int32
		fmt.Sscan(v, &functionID)
		f.FunctionID = &functionID
	}

	err = f.Save(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, f, nil)

}
