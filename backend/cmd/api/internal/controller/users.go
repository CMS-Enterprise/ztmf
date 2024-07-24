package controller

import (
	"context"
	"fmt"

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

func SaveUserFismaSystems(ctx context.Context, userid string, fismasystemids []int32) (*model.User, error) {
	user := auth.UserFromContext(ctx)
	if !user.IsAdmin() {
		return nil, &ForbiddenError{}
	}

	if len(fismasystemids) < 1 {
		return nil, &InvalidInputError{
			field: "fismasystemids",
			value: fmt.Sprintf("%v", fismasystemids),
		}
	}

	err := model.CreateUserFismaSystem(ctx, userid, fismasystemids)
	if err != nil {
		return nil, err
	}

	return model.FindUserById(ctx, graphql.ID(userid))
}
