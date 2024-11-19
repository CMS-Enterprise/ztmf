package model

import (
	"net/mail"
	"regexp"
)

// use of map enables O(1) vs O(N) as would be the case with slices.Contains([]string)
// it also avoids the complexity of using constants as enums
var roles = map[string]interface{}{
	"ADMIN": nil, // the value isn't used, only the ok check value is
	"ISSO":  nil,
	"ISSM":  nil,
}

var datacenterenvironments = map[string]interface{}{
	"Other":          nil,
	"SaaS":           nil,
	"CMS-Cloud-AWS":  nil,
	"CMSDC":          nil,
	"CMS-Cloud-MAG":  nil,
	"AWS":            nil,
	"OPDC":           nil,
	"DECOMMISSIONED": nil,
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

func isValidDataCenterEnvironment(d string) bool {
	_, ok := datacenterenvironments[d]
	return ok
}

func isValidIntID(ID any) bool {
	switch ID.(type) {
	case int32, *int32:
		return true
	}
	return false
}
