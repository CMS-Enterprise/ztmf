package graph

import (
	"context"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

// SaveFunctionScoreResponse represents the code and message returned
type SaveFunctionScoreResponse struct {
	Response
	FunctionScore *model.FunctionScore
}

func (r *RootResolver) SaveFunctionScore(ctx context.Context, args struct {
	Scoreid       *graphql.ID
	Fismasystemid int32
	Functionid    int32
	Score         float64
	Notes         *string
}) *SaveFunctionScoreResponse {
	res := SaveFunctionScoreResponse{}
	functionscore, err := controller.SaveFunctionScore(ctx, args.Scoreid, args.Fismasystemid, args.Functionid, args.Score, args.Notes)
	if err != nil {
		res.Message = err.Error()
		switch err.(type) {
		case *controller.ForbiddenError:
			res.Code = 403
		case *controller.InvalidInputError:
			res.Code = 400
		default:
			res.Code = 500
		}
		return &res
	}

	res.Code = http.StatusOK
	res.Message = "OK"
	res.FunctionScore = functionscore
	return &res
}
