package errors

import (
	"testing"
	"time"
)

func TestCapabilityErrorCodes(t *testing.T) {
	if got := HTTPStatus(CodeUserNotFound); got != 404 {
		t.Errorf("HTTPStatus(CodeUserNotFound) = %d, want 404", got)
	}
	if got := HTTPStatus(CodeQuotaExceeded); got != 403 {
		t.Errorf("HTTPStatus(CodeQuotaExceeded) = %d, want 403", got)
	}
	if got := AccountLocked(90 * time.Second); got.Code != CodeForbidden {
		t.Errorf("AccountLocked code = %d, want %d", got.Code, CodeForbidden)
	}
}
