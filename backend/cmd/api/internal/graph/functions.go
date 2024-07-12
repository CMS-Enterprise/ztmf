package graph

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func (r *RootResolver) Functions(ctx context.Context) ([]*model.Function, error) {
	return controller.ListFunctions(ctx)
}

// resolver for graph entry from root
func (r *RootResolver) Function(ctx context.Context, args struct{ Functionid graphql.ID }) (*model.Function, error) {
	return controller.GetFunction(ctx, args.Functionid)
}
