package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jimu/internal/capabilities/mfa/application"
	mfadomain "jimu/internal/capabilities/mfa/domain"
	"jimu/internal/capabilities/mfa/totp"
	apperrors "jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 测试替身 ----

type fakeMFARepo struct {
	records map[uint64]*mfadomain.UserMFA
}

func newFakeMFARepo() *fakeMFARepo { return &fakeMFARepo{records: map[uint64]*mfadomain.UserMFA{}} }

func (r *fakeMFARepo) FindByUser(_ context.Context, userID uint64) (*mfadomain.UserMFA, error) {
	rec, ok := r.records[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return rec, nil
}

func (r *fakeMFARepo) UpsertSecret(_ context.Context, tenantID, userID uint64, secret string, enabled bool) error {
	r.records[userID] = &mfadomain.UserMFA{ID: userID, TenantID: tenantID, UserID: userID, TOTPSecret: secret, TOTPEnabled: enabled}
	return nil
}

func (r *fakeMFARepo) Clear(_ context.Context, userID uint64) error {
	delete(r.records, userID)
	return nil
}

type fakeDeviceRepo struct {
	devices []*mfadomain.TrustedDevice
}

func newFakeDeviceRepo() *fakeDeviceRepo { return &fakeDeviceRepo{} }

func (r *fakeDeviceRepo) Create(_ context.Context, d *mfadomain.TrustedDevice) error {
	d.ID = uint64(len(r.devices) + 1)
	r.devices = append(r.devices, d)
	return nil
}
func (r *fakeDeviceRepo) FindByTokenHash(context.Context, string) (*mfadomain.TrustedDevice, error) {
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeDeviceRepo) Touch(context.Context, uint64, time.Time) error { return nil }
func (r *fakeDeviceRepo) ListByUser(_ context.Context, _, userID uint64) ([]mfadomain.TrustedDevice, error) {
	var out []mfadomain.TrustedDevice
	for _, d := range r.devices {
		if d.UserID == userID {
			out = append(out, *d)
		}
	}
	return out, nil
}
func (r *fakeDeviceRepo) Delete(context.Context, uint64, uint64, uint64) error    { return nil }
func (r *fakeDeviceRepo) DeleteAllByUser(context.Context, uint64) error           { return nil }
func (r *fakeDeviceRepo) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }

func newMFAHandler() (*MFAHandler, *fakeMFARepo, *fakeDeviceRepo) {
	mfaRepo := newFakeMFARepo()
	devices := newFakeDeviceRepo()
	return NewMFAHandler(application.NewMFAService(mfaRepo, nil, devices, 30, "jimu")), mfaRepo, devices
}

// invoke 直接调用处理器，可选注入 user_id 与 validated_req。
func invoke(t *testing.T, method, target, body string, userID uint64, req any, fn func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := func(c *gin.Context) {
		if userID != 0 {
			c.Set("user_id", userID)
		}
		if req != nil {
			c.Set("validated_req", req)
		}
		fn(c)
	}
	// 同时注册带参数路径，便于覆盖 :id 场景
	r.Handle(method, "/x", handler)
	r.Handle(method, "/x/:id", handler)
	httpReq := httptest.NewRequest(method, target, strings.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httpReq)
	return w
}

func bodyCode(t *testing.T, w *httptest.ResponseRecorder) float64 {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	code, _ := body["code"].(float64)
	return code
}

// ---- 用例 ----

func TestHandlersRequireAuthentication(t *testing.T) {
	h, _, _ := newMFAHandler()
	cases := []struct {
		name string
		req  any
		fn   func(*gin.Context)
	}{
		{"setup", nil, h.SetupTOTP},
		{"enable", &enableTOTPRequest{Code: "123456"}, h.EnableTOTP},
		{"disable", &disableTOTPRequest{Code: "123456"}, h.DisableTOTP},
		{"list devices", nil, h.ListDevices},
		{"revoke device", nil, h.RevokeDevice},
		{"revoke all", nil, h.RevokeAllDevices},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := invoke(t, http.MethodPost, "/x", "", 0, tc.req, tc.fn)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Equal(t, float64(apperrors.CodeUnauthorized), bodyCode(t, w))
		})
	}
}

func TestSetupTOTPReturnsSecretAndURI(t *testing.T) {
	h, repo, _ := newMFAHandler()
	w := invoke(t, http.MethodPost, "/x", "", 42, nil, h.SetupTOTP)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	assert.NotEmpty(t, data["secret"])
	assert.Contains(t, data["otpauth_uri"], "otpauth://totp/")
	assert.False(t, repo.records[42].TOTPEnabled, "setup 只存密钥不启用")
}

func TestEnableDisableTOTPHandlers(t *testing.T) {
	h, repo, _ := newMFAHandler()

	// setup 拿到密钥
	w := invoke(t, http.MethodPost, "/x", "", 42, nil, h.SetupTOTP)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	secret := body["data"].(map[string]any)["secret"].(string)

	// enable：错误码 → 401（CodeInvalidMFA）
	w = invoke(t, http.MethodPost, "/x", "", 42, &enableTOTPRequest{Code: "000000"}, h.EnableTOTP)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// enable：正确码 → 200
	w = invoke(t, http.MethodPost, "/x", "", 42, &enableTOTPRequest{Code: totpCode(t, secret)}, h.EnableTOTP)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.True(t, repo.records[42].TOTPEnabled)

	// disable：错误码 → 401
	w = invoke(t, http.MethodPost, "/x", "", 42, &disableTOTPRequest{Code: "000000"}, h.DisableTOTP)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// disable：正确码 → 200
	w = invoke(t, http.MethodPost, "/x", "", 42, &disableTOTPRequest{Code: totpCode(t, secret)}, h.DisableTOTP)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	_, ok := repo.records[42]
	assert.False(t, ok, "disable 应清除记录")
}

func TestDeviceHandlers(t *testing.T) {
	h, _, devices := newMFAHandler()
	require.NoError(t, devices.Create(context.Background(), &mfadomain.TrustedDevice{UserID: 42, TokenHash: "h1"}))

	// 列表
	w := invoke(t, http.MethodGet, "/x", "", 42, nil, h.ListDevices)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 注销单个：非法 id → 400
	w = invoke(t, http.MethodDelete, "/x/abc", "", 42, nil, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		h.RevokeDevice(c)
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 注销单个：合法 id → 200
	w = invoke(t, http.MethodDelete, "/x/1", "", 42, nil, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		h.RevokeDevice(c)
	})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 注销全部
	w = invoke(t, http.MethodDelete, "/x", "", 42, nil, h.RevokeAllDevices)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// TestRegisterMFARoutes MFA 与可信设备端点注册且恰好一次。
func TestRegisterMFARoutes(t *testing.T) {
	h, _, _ := newMFAHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterMFARoutes(r.Group("/auth"), h.service)

	got := map[string]int{}
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path]++
	}
	for _, want := range []string{
		"POST /auth/mfa/setup", "POST /auth/mfa/enable", "POST /auth/mfa/disable",
		"GET /auth/devices", "DELETE /auth/devices", "DELETE /auth/devices/:id",
	} {
		assert.Equal(t, 1, got[want], "路由应恰好注册一次：%s", want)
	}
}

// totpCode 生成当前有效 TOTP 码（测试辅助）。
func totpCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.Code(secret, time.Now(), totp.DefaultPeriod, totp.DefaultDigits)
	require.NoError(t, err)
	return code
}
