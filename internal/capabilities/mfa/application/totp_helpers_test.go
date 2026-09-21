package application

import (
	"testing"
	"time"

	"jimu/internal/capabilities/mfa/totp"
)

func totpCurrentCode(t *testing.T, secret string) string {
	t.Helper()
	return totpCurrentCodeFor(time.Now(), secret)
}

func totpCurrentCodeFor(now time.Time, secret string) string {
	code, _ := totp.Code(secret, now, totp.DefaultPeriod, totp.DefaultDigits)
	return code
}
