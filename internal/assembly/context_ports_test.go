package assembly

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jimu/internal/contract"
)

// fakeMFAVerifier 满足 contract.MFAVerifier，用于端口注册表语义用例。
type fakeMFAVerifier struct{}

func (fakeMFAVerifier) Enabled(context.Context, uint64) (bool, error) { return false, nil }
func (fakeMFAVerifier) VerifyTOTP(context.Context, uint64, uint64, string) error {
	return nil
}
func (fakeMFAVerifier) MaybeIssueDevice(context.Context, uint64, uint64) string { return "" }
func (fakeMFAVerifier) RevokeDevices(context.Context, uint64)                   {}

func TestPortReturnsZeroWhenAbsent(t *testing.T) {
	ctx := newTestContext(t)
	assert.Nil(t, ctx.Port("mfa"))
	ctx.Provide("mfa", fakeMFAVerifier{})
	require.NotNil(t, ctx.Port("mfa"))
	_, ok := ctx.Port("mfa").(contract.MFAVerifier)
	assert.True(t, ok)
}

// TestProvideRejectsDuplicateName 同名端口重复注册必须报错（不静默覆盖）。
func TestProvideRejectsDuplicateName(t *testing.T) {
	ctx := newTestContext(t)
	require.NoError(t, ctx.Provide("mfa", fakeMFAVerifier{}))
	require.Error(t, ctx.Provide("mfa", fakeMFAVerifier{}))
}
