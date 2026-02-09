package handler

type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

const (
	CodeValidation    = "VALIDATION_ERROR"
	CodeNotFound      = "NOT_FOUND"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeForbidden     = "FORBIDDEN"
	CodeConflict      = "CONFLICT"
	CodeInternalError = "INTERNAL_ERROR"
	CodeBadRequest    = "BAD_REQUEST"
)
