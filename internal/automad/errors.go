package automad

import "fmt"

// ErrorCode classifies an APIError so callers can react without string matching.
type ErrorCode string

const (
	// CodeAuth means authentication failed or the session is not valid.
	CodeAuth ErrorCode = "AUTH"
	// CodeValidation means the caller supplied invalid input.
	CodeValidation ErrorCode = "VALIDATION"
	// CodeForbidden means the write guard rejected the action.
	CodeForbidden ErrorCode = "FORBIDDEN"
	// CodeNotFound means the requested resource does not exist.
	CodeNotFound ErrorCode = "NOT_FOUND"
	// CodeUnsupported means the live bridge is not configured.
	CodeUnsupported ErrorCode = "UNSUPPORTED"
	// CodeUpstream means Automad returned an error we could not classify.
	CodeUpstream ErrorCode = "UPSTREAM"
	// CodeTransport means the request could not be completed (network, timeout).
	CodeTransport ErrorCode = "TRANSPORT"
)

// APIError is a classified error from the Automad bridge.
type APIError struct {
	Code    ErrorCode
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func newError(code ErrorCode, format string, args ...any) *APIError {
	return &APIError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// validationError is a convenience for the common input-validation case.
func validationError(format string, args ...any) *APIError {
	return newError(CodeValidation, format, args...)
}
