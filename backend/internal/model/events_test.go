package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserFromContext_NilUser(t *testing.T) {
	ctx := context.Background()
	user := UserFromContext(ctx)
	assert.Nil(t, user, "UserFromContext should return nil when no user is in context")
}

func TestUserToContext_RoundTrip(t *testing.T) {
	ctx := context.Background()
	original := &User{
		UserID:   "test-id",
		Email:    "test@example.com",
		FullName: "Test User",
		Role:     "ADMIN",
	}

	ctx = UserToContext(ctx, original)
	retrieved := UserFromContext(ctx)

	assert.NotNil(t, retrieved)
	assert.Equal(t, original.UserID, retrieved.UserID)
	assert.Equal(t, original.Email, retrieved.Email)
	assert.Equal(t, original.Role, retrieved.Role)
}

func TestUserToContext_Overwrite(t *testing.T) {
	ctx := context.Background()

	first := &User{UserID: "first", Role: "ADMIN"}
	ctx = UserToContext(ctx, first)

	second := &User{UserID: "second", Role: "READONLY_ADMIN"}
	ctx = UserToContext(ctx, second)

	retrieved := UserFromContext(ctx)
	assert.Equal(t, "second", retrieved.UserID)
	assert.Equal(t, "READONLY_ADMIN", retrieved.Role)
}
