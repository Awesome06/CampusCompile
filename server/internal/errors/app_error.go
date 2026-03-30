package errors

import "fmt"

// AppError is the centralized application error structure.
// It wraps an internal error with an HTTP status and a safe client-facing message.
type AppError struct {
	HTTPStatus int
	Internal   error
	ClientMsg  string
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.ClientMsg, e.Internal)
	}
	return e.ClientMsg
}

// NewAppError constructs a new AppError pointer
func NewAppError(httpStatus int, internal error, clientMsg string) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Internal:   internal,
		ClientMsg:  clientMsg,
	}
}

// Unwrap returns the underlying wrapped error so callers can use errors.Is/As
func (e *AppError) Unwrap() error {
	return e.Internal
}
