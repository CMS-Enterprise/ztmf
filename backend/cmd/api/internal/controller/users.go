package controller

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func ListUsers(ctx context.Context) ([]*model.User, error) {
	user := auth.UserFromContext(ctx)

	if !user.IsSuper() {
		return []*model.User{}, nil
	}

	return model.FindUsers(ctx)
}

func GetUser(ctx context.Context, userid graphql.ID) (*model.User, error) {
	user := auth.UserFromContext(ctx)

	if !user.IsSuper() && user.Userid != userid {
		return nil, nil
	}
	return model.FindUserById(ctx, userid)
}
