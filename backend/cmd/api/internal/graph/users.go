package graph

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func (r *RootResolver) Users(ctx context.Context) ([]*model.User, error) {
	return controller.ListUsers(ctx)
}

func (r *RootResolver) User(ctx context.Context, args struct{ Userid graphql.ID }) (*model.User, error) {
	return controller.GetUser(ctx, args.Userid)
}
