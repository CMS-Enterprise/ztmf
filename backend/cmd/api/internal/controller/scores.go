package controller

import (
	"fmt"
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

// func SaveFunctionScore(ctx context.Context, scoreid *graphql.ID, fismasystemid int32, functionid int32, score float64, notes *string) (*model.FunctionScore, error) {
// 	user := auth.UserFromContext(ctx)
// 	if !user.IsAdmin() && !user.IsAssignedFismaSystem(fismasystemid) {
// 		return nil, &ForbiddenError{}
// 	}

// 	var (
// 		functionscore *model.FunctionScore
// 		err           error
// 	)

// 	if scoreid == nil {
// 		functionscore, err = model.NewFunctionScore(ctx, fismasystemid, functionid, score, notes)
// 	} else {
// 		functionscore, err = model.UpdateFunctionScore(ctx, scoreid, fismasystemid, functionid, score, notes)
// 	}

// 	return functionscore, err
// }
