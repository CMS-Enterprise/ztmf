package controller

import "fmt"

type ForbiddenError struct{}

func (e *ForbiddenError) Error() string {
	return "forbidden"
}

type InvalidInputError struct {
	field string
	value any
}

func (e *InvalidInputError) Error() string {
	return fmt.Sprintf("invalid input for field `%s` with value `%s`", e.field, e.value)
}

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "not found"
}

type ServerError struct{}

func (e *ServerError) Error() string {
	return "server error"
}
