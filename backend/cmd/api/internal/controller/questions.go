package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// TODO: deprecate this in favor of non-nested URIs
//
//	@Summary	List questions relevant to a FISMA system
//	@Tags		fismasystems
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path		int	true	"FISMA system ID"
//	@Success	200				{object}	apiResponse[[]model.Question]
//	@Failure	404				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid}/questions [get]
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

//	@Summary	List all questions
//	@Tags		questions
//	@Produce	json
//	@Security	bearerAuth
//	@Success	200	{object}	apiResponse[[]model.Question]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/questions [get]
func ListQuestions(w http.ResponseWriter, r *http.Request) {
	questions, err := model.FindQuestions(r.Context())
	respond(w, r, questions, err)
}

//	@Summary	Get a question by ID
//	@Tags		questions
//	@Produce	json
//	@Security	bearerAuth
//	@Param		questionid	path		int	true	"Question ID"
//	@Success	200			{object}	apiResponse[model.Question]
//	@Failure	404			{object}	apiResponse[any]
//	@Failure	500			{object}	apiResponse[any]
//	@Router		/questions/{questionid} [get]
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

//	@Summary	Create or update a question
//	@Tags		questions
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		questionid	path		int				false	"Question ID"
//	@Param		body		body		model.Question	true	"Question to save"
//	@Success	201			{object}	apiResponse[model.Question]
//	@Success	204			"No Content"
//	@Failure	400			{object}	apiResponse[any]
//	@Failure	403			{object}	apiResponse[any]
//	@Failure	500			{object}	apiResponse[any]
//	@Router		/questions [post]
//	@Router		/questions/{questionid} [put]
func SaveQuestion(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
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

	q, err = q.Save(r.Context())

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, q, nil)

}
