package interfaces

import (
	"strconv"

	"jimu/internal/modules/user/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *application.UserService
}

func NewUserHandler(service *application.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Create godoc
// @Summary      Create user
// @Description  Create a new user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      application.CreateUserRequest  true  "User info"
// @Security     BearerAuth
// @Success      201  {object}  response.Body
// @Failure      400  {object}  contract.ErrorResponse
// @Failure      409  {object}  contract.ErrorResponse
// @Failure      500  {object}  contract.ErrorResponse
// @Router       /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req application.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	user, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, user)
}

// Get godoc
// @Summary      Get user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  contract.ErrorResponse
// @Failure      404  {object}  contract.ErrorResponse
// @Failure      500  {object}  contract.ErrorResponse
// @Router       /users/{id} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	user, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

// List godoc
// @Summary      List users
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "Page"
// @Param        page_size  query     int     false  "Page size"
// @Param        sort       query     string  false  "Sort field"
// @Param        order      query     string  false  "Sort order"
// @Success      200        {object}  contract.PageResponse
// @Failure      400        {object}  contract.ErrorResponse
// @Failure      500        {object}  contract.ErrorResponse
// @Router       /users [get]
func (h *UserHandler) List(c *gin.Context) {
	var p pagination.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := p.Normalize("id", "username", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if h.service == nil {
		response.Fail(c, errors.New(errors.CodeInternalError, "user service not configured"))
		return
	}
	users, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, users, total, p.Page, p.PageSize)
}

// Update godoc
// @Summary      Update user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                            true  "User ID"
// @Param        body  body      application.UpdateUserRequest  true  "User info"
// @Success      200   {object}  response.Body
// @Failure      400   {object}  contract.ErrorResponse
// @Failure      404   {object}  contract.ErrorResponse
// @Failure      500   {object}  contract.ErrorResponse
// @Router       /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req application.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := h.service.Update(c.Request.Context(), id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete godoc
// @Summary      Delete user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  int  true  "User ID"
// @Success      204
// @Failure      400  {object}  contract.ErrorResponse
// @Failure      500  {object}  contract.ErrorResponse
// @Router       /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}
