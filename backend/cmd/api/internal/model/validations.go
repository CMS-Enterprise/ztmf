package model

import (
	"net/mail"
	"regexp"
)

// use of map enables O(1) vs O(N) as would be the case with slices.Contains([]string)
var roles = map[string]interface{}{
	"ISSO":  nil, // the value isn't used, only the ok check value is
	"ISSM":  nil,
	"ADMIN": nil,
}

var rgxUUID = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func isValidRole(role string) bool {
	_, ok := roles[role]
	return ok
}

func isValidUUID(uuid string) bool {
	return rgxUUID.MatchString(uuid)
}
