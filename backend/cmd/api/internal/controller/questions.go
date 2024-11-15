package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

// TODO: deprecate this in favor of non-nested URIs
func ListFismaSystemQuestions(w http.ResponseWriter, r *http.Request) {
	var fismaSystemID int32
	vars := mux.Vars(r)
	if v, ok := vars["fismasystemid"]; !ok {
		respond(w, r, nil, ErrNotFound)
	} else {
		fmt.Sscan(v, &fismaSystemID)
	}

	questions, err := model.FindQuestionsByFismaSystem(r.Context(), fismaSystemID)
	respond(w, r, questions, err)
}

func ListQuestions(w http.ResponseWriter, r *http.Request) {
	questions, err := model.FindQuestions(r.Context())
	respond(w, r, questions, err)
}

func GetQuestionByID(w http.ResponseWriter, r *http.Request) {
	var questionID int32
	vars := mux.Vars(r)
	if v, ok := vars["questionid"]; !ok {
		respond(w, r, nil, ErrNotFound)
		return
	} else {
		fmt.Sscan(v, &questionID)
	}
	question, err := model.FindQuestionByID(r.Context(), questionID)
	respond(w, r, question, err)
}

func SaveQuestion(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !user.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	q := &model.Question{}

	err := getJSON(r.Body, q)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	vars := mux.Vars(r)
	if v, ok := vars["questionid"]; ok {
		fmt.Sscan(v, &q.QuestionID)
	}

	err = q.Save(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, q, nil)

}
