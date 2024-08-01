package graph

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/graph-gophers/graphql-go"
)

// UserMutationResponse represents the code and message returned
type UserMutationResponse struct {
	Response
	User *model.User
}

func (r *RootResolver) Users(ctx context.Context) ([]*model.User, error) {
	return controller.ListUsers(ctx)
}

func (r *RootResolver) User(ctx context.Context, args struct{ Userid graphql.ID }) (*model.User, error) {
	return controller.GetUser(ctx, args.Userid)
}

func (r *RootResolver) SaveUser(ctx context.Context, args struct {
	Userid   *graphql.ID
	Email    string
	Fullname string
	Role     string
}) *UserMutationResponse {
	res := UserMutationResponse{}
	user, err := controller.SaveUser(ctx, args.Userid, args.Email, args.Fullname, args.Role)
	res.User = user
	if args.Userid == nil {
		res.SetCreated()
	} else {
		res.SetOK()
	}
	res.SetError(err)
	return &res
}

func (r *RootResolver) AssignFismaSystems(ctx context.Context, args struct {
	Userid         string
	Fismasystemids []int32
}) *UserMutationResponse {
	res := UserMutationResponse{}
	user, err := controller.SaveUserFismaSystems(ctx, args.Userid, args.Fismasystemids)
	res.SetOK().SetError(err)
	res.User = user
	return &res
}

func (r *RootResolver) UnassignFismaSystems(ctx context.Context, args struct {
	Userid         string
	Fismasystemids []int32
}) *UserMutationResponse {
	res := UserMutationResponse{}
	user, err := controller.RemoveUserFismaSystems(ctx, args.Userid, args.Fismasystemids)
	res.SetOK().SetError(err)
	res.User = user
	return &res
}
