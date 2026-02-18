package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

func ListScores(w http.ResponseWriter, r *http.Request) {

	var (
		scores []*model.Score
		err    error
	)
	user := model.UserFromContext(r.Context())
	findScoresInput := model.FindScoresInput{}

	if !user.HasAdminRead() {
		findScoresInput.UserID = &user.UserID
	}

	err = decoder.Decode(&findScoresInput, r.URL.Query())
	if err == nil {
		scores, err = model.FindScores(r.Context(), findScoresInput)
	}

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

	if user.IsReadOnlyAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
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
	var (
		aggregate []*model.ScoreAggregate
		err       error
	)

	user := model.UserFromContext(r.Context())
	findScoresInput := model.FindScoresInput{}

	if !user.HasAdminRead() {
		findScoresInput.FismaSystemIDs = user.AssignedFismaSystems
	}

	err = decoder.Decode(&findScoresInput, r.URL.Query())
	if err == nil {
		aggregate, err = model.FindScoresAggregate(r.Context(), findScoresInput)
	}

	respond(w, r, aggregate, err)
}
