package model

import "net/mail"

// use of map enables O(1) vs O(N) as would be the case with slices.Contains([]string)
var roles = map[string]interface{}{
	"ISSO":  nil, // the value isn't used, only the ok check value is
	"ISSM":  nil,
	"ADMIN": nil,
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	if err == nil {
		return true
	}
	return false
}

func isValidRole(role string) bool {
	_, ok := roles[role]
	return ok
}
