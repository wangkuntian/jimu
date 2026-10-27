package auth

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTenantStateChecker struct {
	active bool
	err    error
	calls  []uint64
}

func (f *fakeTenantStateChecker) TenantActive(_ context.Context, tenantID uint64) (bool, error) {
	f.calls = append(f.calls, tenantID)
	return f.active, f.err
}

type fakeUserStateChecker struct {
	active bool
	err    error
	calls  []uint64
}

func (f *fakeUserStateChecker) UserActive(_ context.Context, userID uint64) (bool, error) {
	f.calls = append(f.calls, userID)
	return f.active, f.err
}

func TestAuthMiddlewareChecksTenantState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := New(strings.Repeat("k", 32), "jimu", 30, 7)
	token, err := jwtUtil.GenerateAccess(42, 7, "sid-7")
	require.NoError(t, err)

	tests := []struct {
		name       string
		checker    *fakeTenantStateChecker
		wantStatus int
	}{
		{name: "active tenant", checker: &fakeTenantStateChecker{active: true}, wantStatus: http.StatusNoContent},
		{name: "disabled or deleted tenant", checker: &fakeTenantStateChecker{active: false}, wantStatus: http.StatusUnauthorized},
		{name: "tenant lookup error", checker: &fakeTenantStateChecker{err: stderrors.New("db unavailable")}, wantStatus: http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware(jwtUtil, StateCheckers{Tenant: tc.checker}))
			r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusNoContent) })

			req := httptest.NewRequest(http.MethodGet, "/probe", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Equal(t, []uint64{7}, tc.checker.calls)
		})
	}
}

func TestAuthMiddlewareChecksUserState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := New(strings.Repeat("k", 32), "jimu", 30, 7)
	token, err := jwtUtil.GenerateAccess(42, 7, "sid-7")
	require.NoError(t, err)

	tests := []struct {
		name       string
		checker    *fakeUserStateChecker
		wantStatus int
	}{
		{name: "active user", checker: &fakeUserStateChecker{active: true}, wantStatus: http.StatusNoContent},
		{name: "disabled or deleted user", checker: &fakeUserStateChecker{active: false}, wantStatus: http.StatusUnauthorized},
		{name: "user lookup error", checker: &fakeUserStateChecker{err: stderrors.New("db unavailable")}, wantStatus: http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware(jwtUtil, StateCheckers{User: tc.checker}))
			r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusNoContent) })

			req := httptest.NewRequest(http.MethodGet, "/probe", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Equal(t, []uint64{42}, tc.checker.calls)
		})
	}
}

func TestAuthMiddlewareSkipsTenantCheckForPlatformContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := New(strings.Repeat("k", 32), "jimu", 30, 7)
	token, err := jwtUtil.GenerateAccess(42, 0, "sid-platform")
	require.NoError(t, err)
	tenantChecker := &fakeTenantStateChecker{active: false}
	userChecker := &fakeUserStateChecker{active: true}

	r := gin.New()
	r.Use(AuthMiddleware(jwtUtil, StateCheckers{Tenant: tenantChecker, User: userChecker}))
	r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, tenantChecker.calls)
	assert.Equal(t, []uint64{42}, userChecker.calls)
}
