package controller

// apiResponse documents the standard JSON envelope that every handler returns
// via respond()/respondOK(): a "data" payload on success or an "error" string
// on failure. It exists only so swag annotations can describe the envelope with
// generics, e.g. apiResponse[model.DataCall] or apiResponse[[]model.DataCall].
// The runtime equivalent is the unexported response struct in controller.go.
//
//lint:ignore U1000 referenced only by swag @Success/@Failure annotation comments (not Go code); drives openapi.yaml generation.
type apiResponse[T any] struct {
	Data  T      `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// apiError documents an error that carries a typed code. Used only where a
// client must branch on the code; other errors stay apiResponse[any].
//
//lint:ignore U1000 referenced only by swag @Failure annotation comments.
type apiError struct {
	Error string `json:"error"`
	// REVISION_CONFLICT on the undo 409: refresh history and retry.
	Code string `json:"code"`
}
