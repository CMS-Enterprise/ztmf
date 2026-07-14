package controller

import (
	"errors"
)

var (
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
	ErrServer             = errors.New("server error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrMalformed          = errors.New("json malformed or type mismatch")
	// Returned for gorilla/schema query-param decode failures (e.g.
	// fismasystemid=abc). Client error, so it maps to 400 in sanitizeErr and
	// avoids leaking the library's internal message to the caller. (#420)
	ErrInvalidQueryParam = errors.New("invalid query parameter")
	// lowercase, no period — Go error strings must not be capitalized or end with punctuation
	ErrSelfDelete = errors.New("cannot delete your own account")
)
