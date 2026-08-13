// internal/modules/oauth/domain/binding_test.go
package domain

import (
	"testing"
)

func TestOAuthBindingTableName(t *testing.T) {
	if got := (OAuthBinding{}).TableName(); got != "user_oauth_bindings" {
		t.Fatalf("TableName = %q, want user_oauth_bindings", got)
	}
}
