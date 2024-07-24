package graph

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

// CreateUserResponse represents the code and message returned
type CreateUserResponse struct {
	Response
	User *model.User
}

func (r *RootResolver) Users(ctx context.Context) ([]*model.User, error) {
	return controller.ListUsers(ctx)
}

func (r *RootResolver) User(ctx context.Context, args struct{ Userid graphql.ID }) (*model.User, error) {
	return controller.GetUser(ctx, args.Userid)
}

func (r *RootResolver) CreateUser(ctx context.Context, args struct {
	Email    string
	Fullname string
	Role     string
}) *CreateUserResponse {
	res := CreateUserResponse{}
	user, err := controller.CreateUser(ctx, args.Email, args.Fullname, args.Role)
	res.User = user
	if err != nil {
		res.Message = err.Error()
		switch err.(type) {
		case *controller.ForbiddenError:
			res.Code = 403
		case *controller.InvalidInputError:
			res.Code = 400
		default:
			res.Code = 500
		}
	} else {
		res.Code = 201
		res.Message = "OK"
	}

	return &res
}

// AssignFismaSystemsReponse represents the code and message returned
type AssignFismaSystemsResponse struct {
	Response
	User *model.User
}

func (r *RootResolver) AssignFismaSystems(ctx context.Context, args struct {
	Userid         string
	Fismasystemids []int32
}) (*AssignFismaSystemsResponse, error) {
	res := AssignFismaSystemsResponse{}
	user, err := controller.SaveUserFismaSystems(ctx, args.Userid, args.Fismasystemids)
	res.User = user
	if err != nil {
		res.Message = err.Error()
		switch err.(type) {
		case *controller.ForbiddenError:
			res.Code = 403
		case *controller.InvalidInputError:
			res.Code = 400
		default:
			res.Code = 500
		}
	} else {
		res.Code = 201
		res.Message = "OK"
	}

	return &res, nil
}
