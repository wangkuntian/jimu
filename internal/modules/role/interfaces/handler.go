package interfaces

import (
	"strconv"

	"jimu/internal/modules/role/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service *application.RoleService
}

func NewRoleHandler(service *application.RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

// Create godoc
// @Summary      Create role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      application.CreateRoleRequest  true  "Role info"
// @Success      201   {object}  response.Body
// @Failure      400   {object}  contract.ErrorResponse
// @Failure      409   {object}  contract.ErrorResponse
// @Failure      500   {object}  contract.ErrorResponse
// @Router       /roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	req := c.MustGet("validated_req").(*application.CreateRoleRequest)
	role, err := h.service.Create(c.Request.Context(), *req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, role)
}

// Get godoc
// @Summary      Get role
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Role ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  contract.ErrorResponse
// @Failure      404  {object}  contract.ErrorResponse
// @Failure      500  {object}  contract.ErrorResponse
// @Router       /roles/{id} [get]
func (h *RoleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	role, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, role)
}

// List godoc
// @Summary      List roles
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "Page"
// @Param        page_size  query     int     false  "Page size"
// @Param        sort       query     string  false  "Sort field"
// @Param        order      query     string  false  "Sort order"
// @Success      200        {object}  contract.PageResponse
// @Failure      400        {object}  contract.ErrorResponse
// @Failure      500        {object}  contract.ErrorResponse
// @Router       /roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	p := c.MustGet("validated_query").(*pagination.Pagination)
	if err := p.Normalize("id", "name", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	roles, total, err := h.service.List(c.Request.Context(), *p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, roles, total, p.Page, p.PageSize)
}

// Update godoc
// @Summary      Update role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                            true  "Role ID"
// @Param        body  body      application.UpdateRoleRequest  true  "Role info"
// @Success      200   {object}  response.Body
// @Failure      400   {object}  contract.ErrorResponse
// @Failure      404   {object}  contract.ErrorResponse
// @Failure      409   {object}  contract.ErrorResponse
// @Failure      500   {object}  contract.ErrorResponse
// @Router       /roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req := c.MustGet("validated_req").(*application.UpdateRoleRequest)
	if err := h.service.Update(c.Request.Context(), id, *req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete godoc
// @Summary      Delete role
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  int  true  "Role ID"
// @Success      204
// @Failure      400  {object}  contract.ErrorResponse
// @Failure      500  {object}  contract.ErrorResponse
// @Router       /roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
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

// AssignPermissions godoc
// @Summary      Assign role permissions
// @Tags         roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                                   true  "Role ID"
// @Param        body  body      application.AssignPermissionsRequest  true  "Permission IDs"
// @Success      200   {object}  response.Body
// @Failure      400   {object}  contract.ErrorResponse
// @Failure      500   {object}  contract.ErrorResponse
// @Router       /roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	req := c.MustGet("validated_req").(*application.AssignPermissionsRequest)
	if err := h.service.AssignPermissions(c.Request.Context(), id, req.PermissionIDs); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
