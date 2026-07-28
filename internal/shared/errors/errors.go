package errors

import "fmt"

const (
	CodeOK              = 0
	CodeInvalidParam    = 1001
	CodeUnauthorized    = 1002
	CodeForbidden       = 1003
	CodeNotFound        = 1004
	CodeInternalError   = 1005
	CodeUserNotFound    = 2001
	CodeUserExists      = 2002
	CodeInvalidPassword = 2003
	CodeRoleNotFound    = 2004
)

type AppError struct {
	Code    int
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code int, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}
