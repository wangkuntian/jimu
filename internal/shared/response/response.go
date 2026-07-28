package response

import (
	"errors"
	"net/http"

	appErrs "jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Paginated struct {
	Code     int         `json:"code"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Fail(c *gin.Context, err error) {
	var appErr *appErrs.AppError
	if errors.As(err, &appErr) {
		c.JSON(http.StatusOK, Body{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}
	c.JSON(http.StatusOK, Body{
		Code:    appErrs.CodeInternalError,
		Message: "internal error",
	})
}

func Page(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Paginated{
		Code:     0,
		Message:  "ok",
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
