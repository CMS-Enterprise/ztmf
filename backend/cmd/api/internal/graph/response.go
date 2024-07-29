package graph

import "github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"

// Response represents the results of controller operations where Code is an HTTP status code
// Message is a success or error
type Response struct {
	Code    int32
	Message string
}

// SetError determines the values to set for Code and Message fields based on the provided error
// if err is nil, no fields are set. This allows for SetError to always be called without concern of
// overwriting previously set values which in turn enables efficient use of method chaining
func (r *Response) SetError(err error) *Response {
	if err != nil {
		r.Message = err.Error()
		switch err.(type) {
		case *controller.ForbiddenError:
			r.Code = 403
		case *controller.InvalidInputError:
			r.Code = 400
		default:
			r.Code = 500
		}
	}
	return r
}

// SetCreated sets the code to 201 and Message to CREATED
func (r *Response) SetCreated() *Response {
	r.Code = 201
	r.Message = "CREATED"
	return r
}

// SetCreated sets the code to 200 and Message to OK
func (r *Response) SetOK() *Response {
	r.Code = 200
	r.Message = "OK"
	return r
}
