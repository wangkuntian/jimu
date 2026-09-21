package totp

import (
	"strings"
	"testing"
	"time"
)

// TestCode_RFC6238AppendixB 覆盖 RFC 6238 附录 B sha1 测试向量。
// secret = 12345678901234567890（base32: GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ）
func TestCode_RFC6238AppendixB(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	cases := []struct {
		unix int64
		want string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}
	for _, c := range cases {
		code, err := Code(secret, time.Unix(c.unix, 0), DefaultPeriod, 8)
		if err != nil {
			t.Fatalf("Code() error: %v", err)
		}
		if code != c.want {
			t.Errorf("Code(unix=%d) = %s, want %s", c.unix, code, c.want)
		}
	}
}

func TestCode_DefaultDigits6(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, err := Code(secret, time.Unix(59, 0), DefaultPeriod, 6)
	if err != nil {
		t.Fatalf("Code() error: %v", err)
	}
	if code != "287082" {
		t.Errorf("Code() = %s, want 287082", code)
	}
}

func TestValidate_CurrentWindow(t *testing.T) {
	secret, err := Secret()
	if err != nil {
		t.Fatalf("Secret() error: %v", err)
	}
	now := time.Now()
	code, err := Code(secret, now, DefaultPeriod, DefaultDigits)
	if err != nil {
		t.Fatalf("Code() error: %v", err)
	}
	if !Validate(secret, code, now, DefaultPeriod, DefaultDigits, DefaultSkew) {
		t.Error("Validate() should accept current-window code")
	}
}

func TestValidate_ClockSkew(t *testing.T) {
	secret, _ := Secret()
	// 前一个窗口的 code 应被接受（skew=1）
	prev := time.Now().Add(-time.Duration(DefaultPeriod) * time.Second)
	code, err := Code(secret, prev, DefaultPeriod, DefaultDigits)
	if err != nil {
		t.Fatalf("Code() error: %v", err)
	}
	if !Validate(secret, code, time.Now(), DefaultPeriod, DefaultDigits, 1) {
		t.Error("Validate() should accept one-window-ago code with skew=1")
	}
	// 3 个窗口之前应被拒绝（skew=1）
	old := time.Now().Add(-3 * time.Duration(DefaultPeriod) * time.Second)
	oldCode, _ := Code(secret, old, DefaultPeriod, DefaultDigits)
	if Validate(secret, oldCode, time.Now(), DefaultPeriod, DefaultDigits, 1) {
		t.Error("Validate() should reject code older than skew window")
	}
}

func TestValidate_WrongCode(t *testing.T) {
	secret, _ := Secret()
	if Validate(secret, "000000", time.Now(), DefaultPeriod, DefaultDigits, DefaultSkew) {
		t.Error("Validate() should reject wrong code")
	}
	if Validate("", "123456", time.Now(), DefaultPeriod, DefaultDigits, DefaultSkew) {
		t.Error("Validate() should reject empty secret")
	}
	if Validate(secret, "", time.Now(), DefaultPeriod, DefaultDigits, DefaultSkew) {
		t.Error("Validate() should reject empty code")
	}
}

func TestSecret_ReusableAcrossCalls(t *testing.T) {
	secret, err := Secret()
	if err != nil {
		t.Fatalf("Secret() error: %v", err)
	}
	if len(secret) != 32 {
		t.Errorf("secret length = %d, want 32 (base32 of 20 bytes)", len(secret))
	}
	now := time.Now()
	code, _ := Code(secret, now, DefaultPeriod, DefaultDigits)
	// 校验需要同一个 secret
	if !Validate(secret, code, now, DefaultPeriod, DefaultDigits, DefaultSkew) {
		t.Error("secret should be deterministic for code validation")
	}
}

func TestProvisioningURI(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	uri := ProvisioningURI(secret, "alice", "jimu")
	parts := []string{
		"otpauth://totp/alice",
		"secret=" + secret,
		"issuer=jimu",
		"algorithm=SHA1",
		"digits=6",
		"period=30",
	}
	for _, p := range parts {
		if !strings.Contains(uri, p) {
			t.Errorf("ProvisioningURI() = %q, missing %q", uri, p)
		}
	}
	if !strings.HasPrefix(uri, "otpauth://") {
		t.Errorf("ProvisioningURI() should start with otpauth://, got %q", uri)
	}
}
