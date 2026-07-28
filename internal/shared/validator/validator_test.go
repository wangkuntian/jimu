package validator

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestValidateMobile(t *testing.T) {
	tests := []struct {
		mobile string
		valid  bool
	}{
		{"13800138000", true},
		{"15912345678", true},
		{"12345678901", false}, // 非 1 开头
		{"1380013800", false},  // 少一位
		{"", true},             // 空值由 required 处理
	}

	validate := validator.New()
	_ = validate.RegisterValidation("mobile", validateMobile)

	for _, tt := range tests {
		err := validate.Var(tt.mobile, "mobile")
		if tt.valid && err != nil {
			t.Errorf("expected %q to be valid, got error: %v", tt.mobile, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("expected %q to be invalid", tt.mobile)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"abc12345", true},
		{"Pass1234", true},
		{"12345678", false}, // 纯数字
		{"abcdefgh", false}, // 纯字母
		{"ab12", false},     // 太短
		{"", true},          // 空值由 required 处理
	}

	validate := validator.New()
	_ = validate.RegisterValidation("password", validatePassword)

	for _, tt := range tests {
		err := validate.Var(tt.password, "password")
		if tt.valid && err != nil {
			t.Errorf("expected %q to be valid, got error: %v", tt.password, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("expected %q to be invalid", tt.password)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		username string
		valid    bool
	}{
		{"admin", true},
		{"user_123", true},
		{"abc", false},    // 太短
		{"ab", false},     // 太短
		{"user name", false}, // 含空格
		{"", true},        // 空值由 required 处理
	}

	validate := validator.New()
	_ = validate.RegisterValidation("username", validateUsername)

	for _, tt := range tests {
		err := validate.Var(tt.username, "username")
		if tt.valid && err != nil {
			t.Errorf("expected %q to be valid, got error: %v", tt.username, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("expected %q to be invalid", tt.username)
		}
	}
}
