package interfaces

import (
	"strconv"

	"jimu/internal/capabilities/tenant/application"
	"jimu/internal/kernel/http/middleware"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// PlanHandler 套餐与配额接口
type PlanHandler struct {
	service *application.PlanService
}

func NewPlanHandler(service *application.PlanService) *PlanHandler {
	return &PlanHandler{service: service}
}

// CreatePlan godoc
// @Summary      创建套餐
// @Description  创建租户套餐（资源上限，0 表示不限）。编码全局唯一，统一转小写存储且不可修改。
// @Tags         租户运营
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      application.CreatePlanRequest  true  "套餐信息（编码、名称与各项上限）"
// @Success      201  {object}  response.Body  "创建成功，返回套餐信息"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误"
// @Failure      409  {object}  contract.ErrorResponse  "套餐编码已存在"
// @Router       /tenant-plans [post]
func (h *PlanHandler) CreatePlan(c *gin.Context) {
	req, _ := c.MustGet("validated_req").(*application.CreatePlanRequest)
	plan, err := h.service.Create(c.Request.Context(), *req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, plan)
}

// ListPlans godoc
// @Summary      获取套餐列表
// @Description  分页返回全部套餐定义。
// @Tags         租户运营
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int  false  "页码（默认 1）"
// @Param        page_size  query     int  false  "每页数量（默认 20，最大 100）"
// @Success      200  {object}  contract.PageResponse  "成功，返回分页套餐列表"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误"
// @Router       /tenant-plans [get]
func (h *PlanHandler) ListPlans(c *gin.Context) {
	p, _ := c.MustGet("validated_query").(*pagination.Pagination)
	plans, total, err := h.service.List(c.Request.Context(), *p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, plans, total, p.Page, p.PageSize)
}

// UpdatePlan godoc
// @Summary      更新套餐
// @Description  更新套餐名称与各项上限；编码不可修改。修改后对使用该套餐的租户立即生效。
// @Tags         租户运营
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                            true  "套餐 ID"
// @Param        body  body      application.UpdatePlanRequest  true  "要更新的字段（为空表示不修改）"
// @Success      200  {object}  response.Body  "更新成功"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误"
// @Failure      404  {object}  contract.ErrorResponse  "套餐不存在"
// @Router       /tenant-plans/{id} [put]
func (h *PlanHandler) UpdatePlan(c *gin.Context) {
	id, err := parsePlanID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	req, _ := c.MustGet("validated_req").(*application.UpdatePlanRequest)
	if err := h.service.Update(c.Request.Context(), id, *req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// DeletePlan godoc
// @Summary      删除套餐
// @Description  删除未被任何租户使用的套餐；仍被租户使用时返回 409，避免租户静默失去配额约束。
// @Tags         租户运营
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "套餐 ID"
// @Success      200  {object}  response.Body  "删除成功"
// @Failure      404  {object}  contract.ErrorResponse  "套餐不存在"
// @Failure      409  {object}  contract.ErrorResponse  "套餐仍被租户使用"
// @Router       /tenant-plans/{id} [delete]
func (h *PlanHandler) DeletePlan(c *gin.Context) {
	id, err := parsePlanID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// AssignPlan godoc
// @Summary      为租户分配套餐
// @Description  为指定租户分配套餐；plan_id=0 表示取消套餐（取消后不再受配额限制）。超限时不影响既有数据，只阻止继续创建。
// @Tags         租户运营
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                            true  "租户 ID"
// @Param        body  body      application.AssignPlanRequest  true  "套餐 ID（0=取消套餐）"
// @Success      200  {object}  response.Body  "分配成功"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误"
// @Failure      404  {object}  contract.ErrorResponse  "套餐不存在"
// @Router       /tenants/{id}/plan [put]
func (h *PlanHandler) AssignPlan(c *gin.Context) {
	id, err := parseTenantID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	req, _ := c.MustGet("validated_req").(*application.AssignPlanRequest)
	if err := h.service.Assign(c.Request.Context(), id, req.PlanID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

// Usage godoc
// @Summary      获取当前租户用量
// @Description  返回当前租户的套餐与各资源用量（用户/角色/API Key）及上限，用于容量自查；未分配套餐时 plan 为 null 且不限量。
// @Tags         租户运营
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body  "成功，返回套餐与用量"
// @Failure      401  {object}  contract.ErrorResponse  "未认证"
// @Router       /tenants/usage [get]
func (h *PlanHandler) Usage(c *gin.Context) {
	tenantID := tenant.FromContext(c.Request.Context())
	if tenantID == 0 {
		tenantID = tenant.DefaultTenantID
	}
	usage, err := h.service.Usage(c.Request.Context(), tenantID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, usage)
}

func parsePlanID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New(errors.CodeInvalidParam, "invalid plan id")
	}
	return id, nil
}

// RegisterPlanRoutes 注册套餐与配额路由
func RegisterPlanRoutes(r *gin.RouterGroup, planService *application.PlanService) {
	handler := NewPlanHandler(planService)
	group := r.Group("/tenant-plans")
	{
		group.POST("", middleware.ValidateJSON(&application.CreatePlanRequest{}), handler.CreatePlan)
		group.GET("", middleware.ValidateQuery(&pagination.Pagination{}), handler.ListPlans)
		group.PUT("/:id", middleware.ValidateJSON(&application.UpdatePlanRequest{}), handler.UpdatePlan)
		group.DELETE("/:id", handler.DeletePlan)
	}
	tenants := r.Group("/tenants")
	{
		tenants.PUT("/:id/plan", middleware.ValidateJSON(&application.AssignPlanRequest{}), handler.AssignPlan)
		tenants.GET("/usage", handler.Usage)
	}
}
