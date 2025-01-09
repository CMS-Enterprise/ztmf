package model

import (
	"context"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

// UserFromContext returns the user as stored in the context under the userCtxKey
func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(userCtxKey).(*User)
	return u
}

// UserToContext stores the provided user in the provided context and returns a new context
func UserToContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}
