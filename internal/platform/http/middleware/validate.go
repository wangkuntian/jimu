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

// ValidateJSON 返回一个中间件，自动绑定并验证 JSON 请求体。
// 验证失败时返回 400 + code:1001 并中断；
// 成功时将绑定的对象存入 context key "validated_req"，供 handler 通过 c.MustGet 获取。
func ValidateJSON(dst interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(dst); err != nil {
			response.Fail(c, appErrs.New(appErrs.CodeInvalidParam, translateValidationError(err)))
			c.Abort()
			return
		}
		c.Set("validated_req", dst)
		c.Next()
	}
}

// ValidateQuery 返回一个中间件，自动绑定并验证 Query 参数。
// 成功时将绑定的对象存入 context key "validated_query"。
func ValidateQuery(dst interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindQuery(dst); err != nil {
			response.Fail(c, appErrs.New(appErrs.CodeInvalidParam, translateValidationError(err)))
			c.Abort()
			return
		}
		c.Set("validated_query", dst)
		c.Next()
	}
}

// translateValidationError 将 validator 错误翻译为中文友好消息
func translateValidationError(err error) string {
	var verr validator.ValidationErrors
	if errors.As(err, &verr) && len(verr) > 0 {
		e := verr[0]
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

	msg := err.Error()
	if strings.Contains(msg, "unexpected") || strings.Contains(msg, "invalid character") {
		return "请求体格式错误，请检查 JSON 格式"
	}
	return "请求参数错误"
}
