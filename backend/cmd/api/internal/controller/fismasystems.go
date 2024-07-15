package controller

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func ListFismasystems(ctx context.Context, fismaacronym *string) ([]*model.FismaSystem, error) {
	user := auth.UserFromContext(ctx)
	input := model.FindFismaSystemsInput{
		Fismaacronym: fismaacronym,
	}

	if !user.IsSuper() {
		input.Userid = &user.Userid
	}

	return model.FindFismaSystems(ctx, input)

}

func GetFismasystem(ctx context.Context, fismasystemid graphql.ID) (*model.FismaSystem, error) {
	user := auth.UserFromContext(ctx)
	input := model.FindFismaSystemsInput{
		Fismasystemid: &fismasystemid,
	}

	if !user.IsSuper() {
		input.Userid = &user.Userid
	}

	return model.FindFismaSystem(ctx, input)
}
