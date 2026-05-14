package model

import (
	"net/mail"
	"regexp"
)

// use of map enables O(1) vs O(N) as would be the case with slices.Contains([]string)
// it also avoids the complexity of using constants as enums
//
// New multi-OpDiv role constants live alongside the legacy ADMIN /
// READONLY_ADMIN values during the Stage B -> Stage C transition. The legacy
// values stay valid so controllers that still recognize them keep working
// until Stage C flips the predicate logic. Stage D removes ADMIN and
// READONLY_ADMIN from this map.
//
// Stage D removal checklist (touch every item before dropping the legacy
// keys below, otherwise tests and authn flows break):
//
//  1. THIS FILE: remove the two trailing map entries below.
//  2. backend/internal/model/users.go: drop "ADMIN" / "READONLY_ADMIN" from
//     HasUnscopedRead, IsAdmin, IsReadOnlyAdmin (5 references total).
//  3. backend/cmd/api/internal/auth/middleware.go: local-dev auto-create
//     uses Role: "ADMIN". Change to "OWNER" (or whatever the equivalent
//     unscoped-write tier is at that time).
//  4. backend/_test_data.sql: change role='ADMIN' / 'READONLY_ADMIN' rows
//     to the new tier names.
//  5. backend/_test_data_empire.sql: same.
//  6. backend/emberfall_tests.yml: GET /users/current expects role='ADMIN'
//     for the Test.User fixture. Update to match the new value.
//  7. backend/internal/model/validations_test.go: TestIsValidRole asserts
//     the two legacy roles are still valid. Flip those assertions to
//     false and update the comment block.
//  8. backend/openapi.yaml: User schema role enum drops ADMIN and
//     READONLY_ADMIN; update the description that mentions them.
//  9. Prod data: confirm zero rows still carry the legacy values
//     (SELECT count(*) FROM users WHERE role IN ('ADMIN','READONLY_ADMIN'))
//     before this migration runs. The Stage B swap should have already
//     emptied these, but verify.
// 10. Stage D migration file (0037rolecleanup.go) should add a CHECK
//     constraint or DB-level guard rejecting legacy values post-cleanup.
var roles = map[string]interface{}{
	"OWNER":                nil, // platform / dev team, unscoped across OpDivs
	"HHS_ADMIN":            nil, // department tier, all OpDivs
	"HHS_READONLY_ADMIN":   nil, // department tier, read-only across OpDivs
	"OPDIV_ADMIN":          nil, // single-OpDiv admin, scoped via users_opdivs
	"OPDIV_READONLY_ADMIN": nil, // single-OpDiv read-only, scoped via users_opdivs
	"ISSO":                 nil, // system-scoped via users_fismasystems
	"ISSM":                 nil, // system-scoped via users_fismasystems
	"ADMIN":                nil, // legacy; removed in Stage D
	"READONLY_ADMIN":       nil, // legacy; removed in Stage D
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
