package auth

import (
	"context"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

// UserFromContext returns the user as stored in the context under the userCtxKey
func UserFromContext(ctx context.Context) *model.User {
	u, _ := ctx.Value(userCtxKey).(*model.User)
	return u
}
