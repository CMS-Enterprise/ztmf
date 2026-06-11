package controller

// apiResponse documents the standard JSON envelope that every handler returns
// via respond()/respondOK(): a "data" payload on success or an "error" string
// on failure. It exists only so swag annotations can describe the envelope with
// generics, e.g. apiResponse[model.DataCall] or apiResponse[[]model.DataCall].
// The runtime equivalent is the unexported response struct in controller.go.
type apiResponse[T any] struct {
	Data  T      `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}
