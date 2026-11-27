package errors

import (
	"errors"
	"testing"
)

func TestAppError(t *testing.T) {
	err := New(CodeNotFound, "not found")
	if err.Code != CodeNotFound {
		t.Errorf("expected code %d, got %d", CodeNotFound, err.Code)
	}
	if err.Error() != "[1004] not found" {
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
