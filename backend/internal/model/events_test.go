package model

import (
	"context"
	"encoding/json"
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
		Role:     "OWNER",
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

	first := &User{UserID: "first", Role: "OWNER"}
	ctx = UserToContext(ctx, first)

	second := &User{UserID: "second", Role: "HHS_READONLY_ADMIN"}
	ctx = UserToContext(ctx, second)

	retrieved := UserFromContext(ctx)
	assert.Equal(t, "second", retrieved.UserID)
	assert.Equal(t, "HHS_READONLY_ADMIN", retrieved.Role)
}

// TestQuestionViewInputValidate pins the precondition for recording a view:
// all three identifiers (system, data call, question) are required, and a miss
// surfaces as an *InvalidInputError naming the offending fields so the
// controller returns a 400 rather than writing a malformed event.
func TestQuestionViewInputValidate(t *testing.T) {
	t.Run("AllPresent", func(t *testing.T) {
		in := QuestionViewInput{FismaSystemID: 1, DataCallID: 2, QuestionID: 3}
		assert.NoError(t, in.validate())
	})

	t.Run("AllMissing", func(t *testing.T) {
		err := QuestionViewInput{}.validate()
		iie, ok := err.(*InvalidInputError)
		if assert.True(t, ok, "want *InvalidInputError, got %T", err) {
			assert.Contains(t, iie.Data(), "fismasystemid")
			assert.Contains(t, iie.Data(), "datacallid")
			assert.Contains(t, iie.Data(), "questionid")
		}
	})

	t.Run("PartialMissing", func(t *testing.T) {
		err := QuestionViewInput{FismaSystemID: 1, DataCallID: 2}.validate()
		iie, ok := err.(*InvalidInputError)
		if assert.True(t, ok, "want *InvalidInputError, got %T", err) {
			assert.Contains(t, iie.Data(), "questionid")
			assert.NotContains(t, iie.Data(), "fismasystemid")
		}
	})

	// readonly is not a required field: a false value (an editor view) must
	// still validate, so a zero-value ReadOnly is never treated as missing.
	t.Run("ReadOnlyNotRequired", func(t *testing.T) {
		in := QuestionViewInput{FismaSystemID: 1, DataCallID: 2, QuestionID: 3, ReadOnly: false}
		assert.NoError(t, in.validate())
	})
}

// TestPayloadReadOnlyMarshal pins the wire contract of the readonly flag stored
// on a 'viewed' event's payload: RecordQuestionView stamps a pointer so both
// true and false are serialized (the analytics query needs an explicit editor
// vs viewer signal), while non-view payloads that leave it nil omit the key.
func TestPayloadReadOnlyMarshal(t *testing.T) {
	t.Run("ViewerViewSerializesTrue", func(t *testing.T) {
		ro := true
		b, err := json.Marshal(payload{ReadOnly: &ro})
		assert.NoError(t, err)
		assert.JSONEq(t, `{"readonly":true}`, string(b))
	})

	t.Run("EditorViewSerializesFalse", func(t *testing.T) {
		ro := false
		b, err := json.Marshal(payload{ReadOnly: &ro})
		assert.NoError(t, err)
		assert.JSONEq(t, `{"readonly":false}`, string(b))
	})

	t.Run("NonViewPayloadOmitsReadOnly", func(t *testing.T) {
		b, err := json.Marshal(payload{})
		assert.NoError(t, err)
		assert.JSONEq(t, `{}`, string(b), "a nil readonly pointer is omitted entirely")
	})
}
