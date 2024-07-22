package model

import "fmt"

type InvalidEmailError struct {
	email string
}

func (e *InvalidEmailError) Error() string {
	return fmt.Sprintf("invalid email: %s", e.email)
}

type InvalidRoleError struct {
	role string
}

func (e *InvalidRoleError) Error() string {
	return fmt.Sprintf("invalid role: %s", e.role)
}
