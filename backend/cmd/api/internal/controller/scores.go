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

	vars := mux.Vars(r)

	if v, ok := vars["datacallid"]; ok {
		var dataCallID int32
		fmt.Sscan(v, &dataCallID)
		input.DataCallID = &dataCallID
	}

	if v, ok := vars["fismasystemid"]; ok {
		var fismaSystemID int32
		fmt.Sscan(v, &fismaSystemID)
		input.FismaSystemID = &fismaSystemID
	}

	scores, err := model.FindScores(r.Context(), input)
	respond(w, scores, err)
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
		respond(w, nil, &ForbiddenError{})
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

	respond(w, score, err)
}
