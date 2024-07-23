package controller

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func SaveFunctionScore(ctx context.Context, scoreid *graphql.ID, fismasystemid int32, functionid int32, score float64, notes *string) (*model.FunctionScore, error) {
	user := auth.UserFromContext(ctx)
	if !user.IsAdmin() && !user.IsAssignedFismaSystem(fismasystemid) {
		return nil, &ForbiddenError{}
	}

	var (
		functionscore *model.FunctionScore
		err           error
	)

	if scoreid == nil {
		functionscore, err = model.NewFunctionScore(ctx, fismasystemid, functionid, score, notes)
	} else {
		functionscore, err = model.UpdateFunctionScore(ctx, scoreid, fismasystemid, functionid, score, notes)
	}

	return functionscore, err
}
