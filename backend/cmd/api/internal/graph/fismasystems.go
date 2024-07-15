package graph

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func (r *RootResolver) FismaSystems(ctx context.Context, args struct{ Fismaacronym *string }) ([]*model.FismaSystem, error) {
	return controller.ListFismasystems(ctx, args.Fismaacronym)
}

func (r *RootResolver) FismaSystem(ctx context.Context, args struct{ Fismasystemid graphql.ID }) (*model.FismaSystem, error) {
	return controller.GetFismasystem(ctx, args.Fismasystemid)
}
