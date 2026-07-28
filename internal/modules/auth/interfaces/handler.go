package interfaces

import (
	"jimu/internal/modules/auth/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *application.AuthService
}

func NewAuthHandler(service *application.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
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
// @Description  Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Register info"
// @Success      200   {object}  response.Body
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	user, err := h.service.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	tokenPair, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}
