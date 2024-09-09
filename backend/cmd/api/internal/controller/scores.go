package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListScores(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	input := model.FindScoresInput{}

	if !user.IsAdmin() {
		input.UserID = &user.UserID
	}

	vars := r.URL.Query()
	if vars.Has("datacallid") {
		var dataCallID int32
		fmt.Sscan(vars.Get("datacallid"), &dataCallID)
		input.DataCallID = &dataCallID
	}

	if vars.Has("fismasystemid") {
		var fismaSystemID int32
		fmt.Sscan(vars.Get("fismasystemid"), &fismaSystemID)
		input.FismaSystemID = &fismaSystemID
	}

	scores, err := model.FindScores(r.Context(), input)
	respond(w, r, scores, err)
}

func SaveScore(w http.ResponseWriter, r *http.Request) {
	var (
		scoreID int32
		score   *model.Score
		err     error
	)
	user := auth.UserFromContext(r.Context())
	input := model.SaveScoreInput{}

	err = getJSON(r.Body, &input)
	if err != nil {
		log.Println(err)
	}

	if !user.IsAdmin() && !user.IsAssignedFismaSystem(input.FismaSystemID) {
		respond(w, r, nil, &ForbiddenError{})
		return
	}

	vars := mux.Vars(r)

	if v, ok := vars["scoreid"]; ok {
		fmt.Sscan(v, &scoreID)
		input.ScoreID = &scoreID
	}

	// TODO: distinguish between 200, 201, 204 and dont send body on update
	if input.ScoreID != nil {
		err = model.UpdateScore(r.Context(), input)
	} else {
		score, err = model.CreateScore(r.Context(), input)
	}

	respond(w, r, score, err)
}

func GetScoresAggregate(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	input := model.FindScoresInput{}

	if !user.IsAdmin() {
		input.FismaSystemIDs = user.AssignedFismaSystems
	}

	vars := r.URL.Query()
	if vars.Has("datacallid") {
		var dataCallID int32
		fmt.Sscan(vars.Get("datacallid"), &dataCallID)
		input.DataCallID = &dataCallID
	}

	aggregate, err := model.FindScoresAggregate(r.Context(), input)

	respond(w, r, aggregate, err)
}
