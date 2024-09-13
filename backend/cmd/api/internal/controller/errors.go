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
)
