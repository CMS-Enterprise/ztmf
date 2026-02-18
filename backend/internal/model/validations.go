package model

import (
	"net/mail"
	"regexp"
)

// use of map enables O(1) vs O(N) as would be the case with slices.Contains([]string)
// it also avoids the complexity of using constants as enums
var roles = map[string]interface{}{
	"ADMIN":          nil, // the value isn't used, only the ok check value is
	"READONLY_ADMIN": nil,
	"ISSO":           nil,
	"ISSM":           nil,
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
var rgxUUIDNoDashes = regexp.MustCompile("^[a-fA-F0-9]{32}$")
var rgxDash = regexp.MustCompile("-+")

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func isValidRole(role string) bool {
	_, ok := roles[role]
	return ok
}

// for some reason HHS started removing the dashes from UUID, so some records in fismasystems have dashes and some dont
// while all records in users table still have them
func isValidUUID(uuid string) bool {
	if !rgxDash.MatchString(uuid) {
		return rgxUUIDNoDashes.MatchString(uuid)
	}
	return rgxUUID.MatchString(uuid)
}

func isValidDataCenterEnvironment(d string) bool {
	_, ok := datacenterenvironments[d]
	return ok
}

func isValidIntID(ID any) bool {
	var i int32
	switch ID := ID.(type) {
	case int32:
		i = ID
	case *int32:
		if ID != nil {
			i = *ID
		}
	}
	return i > 0
}
