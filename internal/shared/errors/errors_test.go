package errors

import (
	"errors"
	"testing"
)

func TestAppError(t *testing.T) {
	err := New(CodeUserNotFound, "user not found")
	if err.Code != CodeUserNotFound {
		t.Errorf("expected code %d, got %d", CodeUserNotFound, err.Code)
	}
	if err.Error() != "[2001] user not found" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}

func TestAppErrorWithCause(t *testing.T) {
	cause := errors.New("db connection failed")
	err := Wrap(CodeInternalError, "internal error", cause)
	if err.Cause != cause {
		t.Error("expected cause to be preserved")
	}
	if !errors.Is(err, cause) {
		t.Error("expected Unwrap to work with errors.Is")
	}
}
