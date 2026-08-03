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
		})
		return
	}
	c.JSON(http.StatusInternalServerError, Body{
		Code:      appErrs.CodeInternalError,
		Message:   "internal error",
		RequestID: requestID(c),
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

func StatusForCode(code int) int {
	switch code {
	case appErrs.CodeOK:
		return http.StatusOK
	case appErrs.CodeInvalidParam:
		return http.StatusBadRequest
	case appErrs.CodeUnauthorized, appErrs.CodeInvalidCredentials, appErrs.CodeInvalidPassword:
		return http.StatusUnauthorized
	case appErrs.CodeForbidden:
		return http.StatusForbidden
	case appErrs.CodeNotFound, appErrs.CodeUserNotFound, appErrs.CodeRoleNotFound:
		return http.StatusNotFound
	case appErrs.CodeConflict, appErrs.CodeUserExists:
		return http.StatusConflict
	case appErrs.CodeRateLimited:
		return http.StatusTooManyRequests
	case appErrs.CodeTimeout:
		return http.StatusGatewayTimeout
	case appErrs.CodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func requestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.GetString("request_id")
}
