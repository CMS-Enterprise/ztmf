package controller

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func ListFunctions(ctx context.Context) ([]*model.Function, error) {
	return model.FindFunctions(ctx)
}

func GetFunction(ctx context.Context, functionid graphql.ID) (*model.Function, error) {
	return model.FindFunctionById(ctx, functionid)
}
