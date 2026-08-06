package middleware

import (
	"errors"
	"fmt"
	"strings"

	appErrs "jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// fieldError 单个字段校验错误
type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateJSON 返回一个中间件，自动绑定并验证 JSON 请求体。
// 验证失败时返回 400 + code:1001，并在 details 中附带字段级错误；
// 成功时将绑定的对象存入 context key "validated_req"，供 handler 通过 c.MustGet 获取。
func ValidateJSON(dst interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(dst); err != nil {
			details := translateValidationDetails(err)
			msg := "请求参数错误"
			if len(details) > 0 {
				msg = details[0].Message
			}
			response.FailWithDetails(c, appErrs.New(appErrs.CodeInvalidParam, msg), details)
			c.Abort()
			return
		}
		c.Set("validated_req", dst)
		c.Next()
	}
}

// ValidateQuery 返回一个中间件，自动绑定并验证 Query 参数。
// 验证失败时返回 400 + code:1001，并在 details 中附带字段级错误；
// 成功时将绑定的对象存入 context key "validated_query"。
func ValidateQuery(dst interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindQuery(dst); err != nil {
			details := translateValidationDetails(err)
			msg := "请求参数错误"
			if len(details) > 0 {
				msg = details[0].Message
			}
			response.FailWithDetails(c, appErrs.New(appErrs.CodeInvalidParam, msg), details)
			c.Abort()
			return
		}
		c.Set("validated_query", dst)
		c.Next()
	}
}

// translateValidationDetails 将 validator 错误翻译为字段级错误列表
func translateValidationDetails(err error) []fieldError {
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		details := make([]fieldError, 0, len(verr))
		for _, e := range verr {
			details = append(details, fieldError{
				Field:   e.Field(),
				Message: translateValidationMessage(e),
			})
		}
		return details
	}
	if strings.Contains(err.Error(), "unexpected") || strings.Contains(err.Error(), "invalid character") {
		return []fieldError{{Field: "body", Message: "请求体格式错误，请检查 JSON 格式"}}
	}
	return nil
}

// translateValidationMessage 将单个 validator 字段错误翻译为中文友好消息
func translateValidationMessage(e validator.FieldError) string {
	field := e.Field()
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s 不能为空", field)
	case "min":
		return fmt.Sprintf("%s 长度不能少于 %s 个字符", field, e.Param())
	case "max":
		return fmt.Sprintf("%s 长度不能超过 %s 个字符", field, e.Param())
	case "email":
		return fmt.Sprintf("%s 邮箱格式不正确", field)
	case "len":
		return fmt.Sprintf("%s 长度必须为 %s 个字符", field, e.Param())
	case "gte":
		return fmt.Sprintf("%s 不能小于 %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s 不能大于 %s", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s 必须是以下值之一: %s", field, strings.ReplaceAll(e.Param(), " ", ", "))
	case "mobile":
		return fmt.Sprintf("%s 手机号格式不正确", field)
	case "password":
		return fmt.Sprintf("%s 密码必须为 8-32 位且包含字母和数字", field)
	case "username":
		return fmt.Sprintf("%s 用户名必须为 4-20 位字母、数字或下划线", field)
	case "idcard":
		return fmt.Sprintf("%s 身份证号格式不正确", field)
	default:
		return fmt.Sprintf("%s 校验失败: %s", field, e.Tag())
	}
}
