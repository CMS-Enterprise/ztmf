package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/gorilla/mux"
)

func ListScores(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
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
		err     error
	)

	user := model.UserFromContext(r.Context())
	score := &model.Score{}

	err = getJSON(r.Body, &score)
	if err != nil {
		log.Println(err)
	}

	if !user.IsAdmin() && !user.IsAssignedFismaSystem(score.FismaSystemID) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)

	if v, ok := vars["scoreid"]; ok {
		fmt.Sscan(v, &scoreID)
		score.ScoreID = scoreID
	}

	score, err = score.Save(r.Context())

	respond(w, r, score, err)
}

func GetScoresAggregate(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
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
