package controller

// TODO: reimplement for REST
// import "net/mail"

// // use of map enables O(1) vs O(N) as would be the case with slices.Contains([]string)
// var roles = map[string]bool{
// 	"ISSO":  true, // bool value isnt used, only the ok value is
// 	"ISSM":  true,
// 	"ADMIN": true,
// }

// func validateEmail(email string) error {
// 	_, err := mail.ParseAddress(email)
// 	if err != nil {
// 		return &InvalidInputError{field: "email", value: email}
// 	}
// 	return nil
// }

// func validateRole(role string) error {
// 	if _, ok := roles[role]; !ok {
// 		return &InvalidInputError{field: "role", value: role}
// 	}
// 	return nil
// }
