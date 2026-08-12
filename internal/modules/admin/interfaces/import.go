package interfaces

import (
	"bytes"
	"strconv"

	"jimu/internal/modules/admin/application"
	"jimu/internal/platform/importer"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// AdminImportHandler 数据导入 handler
type AdminImportHandler struct {
	service *application.ImportService
}

// NewAdminImportHandler 创建数据导入 handler
func NewAdminImportHandler(service *application.ImportService) *AdminImportHandler {
	return &AdminImportHandler{service: service}
}

// Preview 预览导入结果（校验不落库）
func (h *AdminImportHandler) Preview(c *gin.Context) {
	format, file, err := bindImportFile(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	result, err := h.service.Preview(c.Request.Context(), format, file, c.DefaultQuery("type", "users"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// Import 执行导入
func (h *AdminImportHandler) Import(c *gin.Context) {
	format, file, err := bindImportFile(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	createdBy, _ := c.Get("user_id")
	byID, _ := createdBy.(uint64)
	fileHeader, _ := c.FormFile("file")
	filename := ""
	if fileHeader != nil {
		filename = fileHeader.Filename
	}
	result, job, err := h.service.Import(c.Request.Context(), format, file, c.DefaultQuery("type", "users"), byID, filename)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if job == nil {
		// 校验失败，未创建任务
		response.OK(c, gin.H{"validation_error": true, "result": result})
		return
	}
	response.OK(c, gin.H{"import_job_id": job.ID, "result": result})
}

// Get 查询导入任务状态
func (h *AdminImportHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	job, err := h.service.GetImportJob(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, job)
}

// Template 下载导入模板（CSV）
func (h *AdminImportHandler) Template(c *gin.Context) {
	csvData := []byte("username,password,email\n")
	c.Header("Content-Disposition", `attachment; filename="import_users_template.csv"`)
	c.Data(200, "text/csv", csvData)
}

// bindImportFile 解析上传文件并返回格式与内容
func bindImportFile(c *gin.Context) (importer.Format, *bytes.Buffer, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return "", nil, errors.New(errors.CodeInvalidParam, "missing file")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return "", nil, errors.New(errors.CodeInvalidParam, "open file failed")
	}
	defer func() { _ = file.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(file); err != nil {
		return "", nil, errors.New(errors.CodeInvalidParam, "read file failed")
	}
	format := importer.FormatCSV
	switch fileHeader.Header.Get("Content-Type") {
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/octet-stream":
		if len(fileHeader.Filename) > 0 && fileHeader.Filename[len(fileHeader.Filename)-5:] == ".xlsx" {
			format = importer.FormatExcel
		}
	}
	return format, &buf, nil
}
