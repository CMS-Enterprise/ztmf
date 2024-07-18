package controller

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

func ListUsers(ctx context.Context) ([]*model.User, error) {
	user := auth.UserFromContext(ctx)

	if !user.IsAdmin() {
		return []*model.User{}, nil
	}

	return model.FindUsers(ctx)
}

func GetUser(ctx context.Context, userid graphql.ID) (*model.User, error) {
	user := auth.UserFromContext(ctx)

	if !user.IsAdmin() && user.Userid != userid {
		return nil, nil
	}
	return model.FindUserById(ctx, userid)
}

func CreateUser(ctx context.Context, email, fullname, role string) (*model.User, error) {
	currentUser := auth.UserFromContext(ctx)

	if !currentUser.IsAdmin() {
		return nil, &ForbiddenError{}
	}

	if err := validateEmail(email); err != nil {
		return nil, err
	}

	if err := validateRole(role); err != nil {
		return nil, err
	}

	return model.NewUser(ctx, email, fullname, role)

}
