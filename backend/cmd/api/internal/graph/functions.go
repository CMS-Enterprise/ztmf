package graph

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func (r *RootResolver) Functions(ctx context.Context) ([]*model.Function, error) {
	// TODO: check ACL
	return model.FindFunctions(ctx)

}

// resolver for graph entry from root
func (r *RootResolver) Function(ctx context.Context, args struct{ Functionid graphql.ID }) (*model.Function, error) {
	// TODO: check ACL
	return model.FindFunctionById(ctx, args.Functionid)
}
