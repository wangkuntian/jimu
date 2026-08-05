package interfaces

import (
	"strings"
	"time"

	"jimu/internal/config"
	"jimu/internal/modules/auth/application"
	platformauth "jimu/internal/platform/auth"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *application.AuthService
	cfg     config.AuthConfig
	limiter *platformauth.Limiter
}

func NewAuthHandler(service *application.AuthService, cfg config.AuthConfig, limiter *platformauth.Limiter) *AuthHandler {
	return &AuthHandler{service: service, cfg: cfg, limiter: limiter}
}

// Login godoc
// @Summary      User login
// @Description  Login with username and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Login info"
// @Success      200   {object}  response.Body
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	req := c.MustGet("validated_req").(*loginRequest)
	if !h.allow(c, "login", "ip:"+c.ClientIP(), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	if !h.allow(c, "login", "username:"+normalizeUsername(req.Username), h.cfg.LoginRateLimit, time.Duration(h.cfg.LoginRateWindowSec)*time.Second) {
		return
	}
	tokenPair, err := h.service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}

// Register godoc
// @Summary      Register user
// @Description  Register a new user when public registration is enabled
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Register info"
// @Success      200   {object}  response.Body
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	req := c.MustGet("validated_req").(*loginRequest)
	if !h.allow(c, "register", "ip:"+c.ClientIP(), h.cfg.RegisterRateLimit, time.Duration(h.cfg.RegisterRateWindowSec)*time.Second) {
		return
	}
	user, err := h.service.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Get new access token using refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      refreshRequest  true  "Refresh token"
// @Success      200   {object}  response.Body
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	req := c.MustGet("validated_req").(*refreshRequest)
	tokenPair, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}

// Logout godoc
// @Summary      Logout current session
// @Description  Revoke the current refresh session
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, sessionID, ok := authContext(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid session"))
		return
	}
	if err := h.service.Logout(c.Request.Context(), userID, sessionID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

// LogoutAll godoc
// @Summary      Logout all sessions
// @Description  Revoke all refresh sessions for the current user
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body
// @Router       /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID, _, ok := authContext(c)
	if !ok {
		response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid session"))
		return
	}
	if err := h.service.LogoutAll(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

func authContext(c *gin.Context) (uint64, string, bool) {
	userID, ok := c.Get("user_id")
	if !ok {
		return 0, "", false
	}
	id, ok := userID.(uint64)
	if !ok {
		return 0, "", false
	}
	return id, c.GetString("session_id"), c.GetString("session_id") != ""
}

func (h *AuthHandler) allow(c *gin.Context, scope, key string, limit int, window time.Duration) bool {
	if h.limiter == nil {
		return true
	}
	ok, err := h.limiter.Allow(c.Request.Context(), scope, key, limit, window)
	if err != nil && ok {
		return true
	}
	if err != nil || !ok {
		response.Fail(c, errors.New(errors.CodeRateLimited, "too many requests"))
		return false
	}
	return true
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}
