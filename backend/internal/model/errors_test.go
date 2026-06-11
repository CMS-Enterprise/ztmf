package model

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// trapError converts driver errors into the package's generic errors. The 28P01
// case is the important regression guard: it used to log.Fatal (killing the
// process on every RDS secret rotation). db.Conn now refreshes and retries on
// 28P01, so a 28P01 reaching trapError must return a normal connection error
// rather than exiting - if this test even completes, the process did not die.
func TestTrapError(t *testing.T) {
	tests := []struct {
		name string
		in   error
		want error
	}{
		{"nil passes through", nil, nil},
		{"no rows", pgx.ErrNoRows, ErrNoData},
		{"too many rows", pgx.ErrTooManyRows, ErrTooMuchData},
		{"foreign key violation", &pgconn.PgError{Code: "23503"}, ErrNoReference},
		{"auth failure is not fatal", &pgconn.PgError{Code: "28P01", Message: "password authentication failed"}, ErrDbConnection},
		{"wrapped auth failure is not fatal", fmt.Errorf("connect: %w", &pgconn.PgError{Code: "28P01"}), ErrDbConnection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trapError(tt.in)
			if tt.want == nil {
				assert.NoError(t, got)
				return
			}
			assert.ErrorIs(t, got, tt.want)
		})
	}
}

// A unique violation is wrapped with detail, so it satisfies errors.Is against
// ErrNotUnique rather than being equal to it.
func TestTrapError_UniqueViolation(t *testing.T) {
	got := trapError(&pgconn.PgError{Code: "23505", Detail: "Key (email)=(x@y.z) already exists."})
	assert.ErrorIs(t, got, ErrNotUnique)
}
