package http

const (
	// errBadRequestBody is the response message to invalid data in client requests.
	errBadRequestBody string = "bad request body"
)

// errResponse is a wrapper for returning JSON error messages.
type errResponse struct {
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}
