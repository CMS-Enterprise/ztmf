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
