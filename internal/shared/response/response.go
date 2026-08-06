package response

import (
	"errors"
	"net/http"

	appErrs "jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}

type Paginated struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Total     int64       `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	RequestID string      `json:"request_id,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: requestID(c),
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Body{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: requestID(c),
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Fail(c *gin.Context, err error) {
	failWithDetails(c, err, nil)
}

// FailWithDetails 返回带字段级错误详情的失败响应
func FailWithDetails(c *gin.Context, err error, details interface{}) {
	failWithDetails(c, err, details)
}

func failWithDetails(c *gin.Context, err error, details interface{}) {
	var appErr *appErrs.AppError
	if errors.As(err, &appErr) {
		message := appErr.Message
		if appErr.Code == appErrs.CodeInternalError {
			message = "internal error"
		}
		c.JSON(StatusForCode(appErr.Code), Body{
			Code:      appErr.Code,
			Message:   message,
			RequestID: requestID(c),
			Details:   details,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, Body{
		Code:      appErrs.CodeInternalError,
		Message:   "internal error",
		RequestID: requestID(c),
		Details:   details,
	})
}

func Page(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Paginated{
		Code:      0,
		Message:   "ok",
		Data:      data,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		RequestID: requestID(c),
	})
}

// StatusForCode 返回错误码对应的 HTTP 状态码
func StatusForCode(code int) int {
	return appErrs.HTTPStatus(code)
}

func requestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.GetString("request_id")
}
