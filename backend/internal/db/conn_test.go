package db

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// isAuthError is what decides whether Conn refreshes the cached secret and
// retries: only a 28P01 (password authentication failed) should trigger the
// re-fetch, and it must still be recognized when pgx wraps the PgError.
func TestIsAuthError(t *testing.T) {
	authErr := &pgconn.PgError{Code: "28P01", Message: `password authentication failed for user "ztmfAdmin"`}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"bare 28P01", authErr, true},
		{"wrapped 28P01", fmt.Errorf("failed to connect: %w", authErr), true},
		{"other pg error (unique violation)", &pgconn.PgError{Code: "23505"}, false},
		{"non-pg error", errors.New("dial tcp: connection refused"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isAuthError(tt.err))
		})
	}
}
