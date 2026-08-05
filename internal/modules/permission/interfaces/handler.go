package interfaces

import (
	"strconv"

	"jimu/internal/modules/permission/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type PermissionHandler struct {
	service *application.PermissionService
}

func NewPermissionHandler(service *application.PermissionService) *PermissionHandler {
	return &PermissionHandler{service: service}
}

// Create godoc
// @Summary      Create permission
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      application.CreatePermissionRequest  true  "Permission info"
// @Success      201   {object}  response.Body
// @Failure      400   {object}  contract.ErrorResponse
// @Failure      409   {object}  contract.ErrorResponse
// @Failure      500   {object}  contract.ErrorResponse
// @Router       /permissions [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	req := c.MustGet("validated_req").(*application.CreatePermissionRequest)
	perm, err := h.service.Create(c.Request.Context(), *req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, perm)
}

// Get godoc
// @Summary      Get permission
// @Tags         permissions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Permission ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  contract.ErrorResponse
// @Failure      404  {object}  contract.ErrorResponse
// @Failure      500  {object}  contract.ErrorResponse
// @Router       /permissions/{id} [get]
func (h *PermissionHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	perm, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, perm)
}

// List godoc
// @Summary      List permissions
// @Tags         permissions
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "Page"
// @Param        page_size  query     int     false  "Page size"
// @Param        sort       query     string  false  "Sort field"
// @Param        order      query     string  false  "Sort order"
// @Success      200        {object}  contract.PageResponse
// @Failure      400        {object}  contract.ErrorResponse
// @Failure      500        {object}  contract.ErrorResponse
// @Router       /permissions [get]
func (h *PermissionHandler) List(c *gin.Context) {
	p := c.MustGet("validated_query").(*pagination.Pagination)
	if err := p.Normalize("id", "name", "resource", "action", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	perms, total, err := h.service.List(c.Request.Context(), *p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, perms, total, p.Page, p.PageSize)
}

// Update godoc
// @Summary      Update permission
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                                  true  "Permission ID"
// @Param        body  body      application.UpdatePermissionRequest  true  "Permission info"
// @Success      200   {object}  response.Body
// @Failure      400   {object}  contract.ErrorResponse
// @Failure      404   {object}  contract.ErrorResponse
// @Failure      409   {object}  contract.ErrorResponse
// @Failure      500   {object}  contract.ErrorResponse
// @Router       /permissions/{id} [put]
func (h *PermissionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req := c.MustGet("validated_req").(*application.UpdatePermissionRequest)
	if err := h.service.Update(c.Request.Context(), id, *req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete godoc
// @Summary      Delete permission
// @Tags         permissions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  int  true  "Permission ID"
// @Success      204
// @Failure      400  {object}  contract.ErrorResponse
// @Failure      500  {object}  contract.ErrorResponse
// @Router       /permissions/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
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
