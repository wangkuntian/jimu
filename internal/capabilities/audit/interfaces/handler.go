package interfaces

import (
	"fmt"
	"strconv"
	"time"

	"jimu/internal/capabilities/audit/application"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	service *application.AuditService
}

func NewAuditHandler(service *application.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

// Verify godoc
// @Summary      校验审计链完整性
// @Description  按 ID 升序重算审计日志的链式哈希（HMAC-SHA256/SHA-256）并检查前后衔接，用于发现篡改或删除。返回参与校验的条目数、未哈希的存量条目数、是否完整、首个异常条目与原因；单租户全量校验时还会比对链头检测末尾条目被删除。
// @Tags         审计日志
// @Produce      json
// @Security     BearerAuth
// @Param        from_id  query     int     false  "起始条目 ID（含），默认从头开始"
// @Param        to_id    query     int     false  "结束条目 ID（含），默认到最后"
// @Param        limit    query     int     false  "最多校验条数（默认 1000）"
// @Success      200      {object}  response.Body{data=application.VerifyResult}  "成功，返回校验结果"
// @Failure      500      {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /audits/verify [get]
func (h *AuditHandler) Verify(c *gin.Context) {
	fromID, _ := strconv.ParseUint(c.Query("from_id"), 10, 64)
	toID, _ := strconv.ParseUint(c.Query("to_id"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))

	result, err := h.service.Verify(c.Request.Context(), tenant.FromContext(c.Request.Context()), fromID, toID, limit)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// Export godoc
// @Summary      导出审计日志
// @Description  按时间范围流式导出当前租户的审计日志（平台级视角导出全部租户）。format=csv（默认，含 UTF-8 BOM）或 json（NDJSON，每行一个对象）；单次最多 50000 条、跨度最多 90 天，超过返回参数错误，请缩小时间范围。
// @Tags         审计日志
// @Produce      text/csv
// @Security     BearerAuth
// @Param        format  query     string  false  "导出格式：csv（默认）或 json"
// @Param        start   query     string  false  "起始时间（RFC3339，默认 7 天前）"
// @Param        end     query     string  false  "结束时间（RFC3339，默认当前）"
// @Success      200     {string}  string  "导出文件内容"
// @Failure      400     {object}  contract.ErrorResponse  "参数错误（格式、时间范围或条数超限）"
// @Failure      500     {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /audits/export [get]
func (h *AuditHandler) Export(c *gin.Context) {
	opts := application.ExportOptions{Format: c.Query("format")}
	if raw := c.Query("start"); raw != "" {
		start, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "start must be RFC3339"))
			return
		}
		opts.Start = start
	}
	if raw := c.Query("end"); raw != "" {
		end, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Fail(c, errors.New(errors.CodeInvalidParam, "end must be RFC3339"))
			return
		}
		opts.End = end
	}

	// 先算出文件名与 Content-Type：数据一旦开始写入就不能再改响应头
	contentType := "text/csv; charset=utf-8"
	if opts.Format == application.ExportFormatJSON {
		contentType = "application/x-ndjson; charset=utf-8"
	}
	filename := fmt.Sprintf("audits_%s.%s", time.Now().UTC().Format("20060102T150405Z"), exportExtension(opts.Format))
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	if opts.Format == application.ExportFormatCSV {
		// BOM：让 Excel 正确识别 UTF-8 中文
		_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	}

	summary, err := h.service.Export(c.Request.Context(), opts, c.Writer)
	if err != nil {
		// 已经开始流式写出时无法改状态码，记录日志即可
		if !c.Writer.Written() {
			response.Fail(c, err)
			return
		}
		c.Error(err) //nolint:errcheck // gin 错误收集，无返回值
		return
	}
	c.Header("X-Export-Rows", strconv.FormatInt(summary.Rows, 10))
}

// exportExtension 导出文件扩展名
func exportExtension(format string) string {
	if format == application.ExportFormatJSON {
		return "ndjson"
	}
	return "csv"
}

// List godoc
// @Summary      获取审计日志列表
// @Description  分页获取系统审计日志。记录用户的操作行为（如登录、创建、修改、删除等），用于安全审计和行为追踪。支持按 ID、创建时间排序。
// @Tags         审计日志
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "页码（默认 1）"
// @Param        page_size  query     int     false  "每页数量（默认 20，最大 100）"
// @Param        sort       query     string  false  "排序字段（支持 id、created_at，默认 id）"
// @Param        order      query     string  false  "排序方向（asc 或 desc，默认 desc）"
// @Success      200        {object}  contract.PageResponse  "成功，返回分页审计日志列表"
// @Failure      400        {object}  contract.ErrorResponse  "参数错误（如无效的排序字段）"
// @Failure      500        {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /audits [get]
func (h *AuditHandler) List(c *gin.Context) {
	p, _ := c.MustGet("validated_query").(*pagination.Pagination)
	if err := p.Normalize("id", "created_at"); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	logs, total, err := h.service.List(c.Request.Context(), *p)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, logs, total, p.Page, p.PageSize)
}

// Get godoc
// @Summary      获取审计日志详情
// @Description  根据审计日志 ID 获取详细信息，包括操作用户、操作类型、资源、详情、IP 地址、HTTP 方法、请求路径和状态码。
// @Tags         审计日志
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "审计日志 ID"
// @Success      200  {object}  response.Body  "成功，返回审计日志详情"
// @Failure      400  {object}  contract.ErrorResponse  "参数错误（ID 格式无效）"
// @Failure      404  {object}  contract.ErrorResponse  "审计日志不存在"
// @Failure      500  {object}  contract.ErrorResponse  "服务器内部错误"
// @Router       /audits/{id} [get]
func (h *AuditHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	log, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, log)
}
