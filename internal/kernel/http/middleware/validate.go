package middleware

import (
	"errors"
	"strings"

	appErrs "jimu/internal/shared/errors"
	"jimu/internal/shared/i18n"
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
			locale := localeOf(c)
			details := translateValidationDetails(err, locale)
			msg := i18n.T("invalid_param", locale)
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
			locale := localeOf(c)
			details := translateValidationDetails(err, locale)
			msg := i18n.T("invalid_param", locale)
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

// localeOf 读取 gin context 中的语言，缺省 zh
func localeOf(c *gin.Context) string {
	if l := c.GetString("locale"); l != "" {
		return l
	}
	return i18n.LangZH
}

// translateValidationDetails 将 validator 错误翻译为字段级错误列表
func translateValidationDetails(err error, locale string) []fieldError {
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		details := make([]fieldError, 0, len(verr))
		for _, e := range verr {
			details = append(details, fieldError{
				Field:   e.Field(),
				Message: translateValidationMessage(e, locale),
			})
		}
		return details
	}
	if strings.Contains(err.Error(), "unexpected") || strings.Contains(err.Error(), "invalid character") {
		return []fieldError{{Field: "body", Message: i18n.T("validation_body_json", locale)}}
	}
	return nil
}

// translateValidationMessage 将单个 validator 字段错误按语言翻译为友好消息
func translateValidationMessage(e validator.FieldError, locale string) string {
	field := e.Field()
	var key string
	switch e.Tag() {
	case "required":
		key = "validation_required"
	case "min":
		key = "validation_min"
	case "max":
		key = "validation_max"
	case "email":
		key = "validation_email"
	case "len":
		key = "validation_len"
	case "gte":
		key = "validation_gte"
	case "lte":
		key = "validation_lte"
	case "oneof":
		return i18n.Tf("validation_oneof", locale, field, strings.ReplaceAll(e.Param(), " ", ", "))
	case "mobile":
		key = "validation_mobile"
	case "password":
		key = "validation_password"
	case "username":
		key = "validation_username"
	case "idcard":
		key = "validation_idcard"
	default:
		return i18n.Tf("validation_default", locale, field, e.Tag())
	}
	// required 只有一个占位符（字段名）；其余规则带参数
	if key == "validation_required" {
		return i18n.Tf(key, locale, field)
	}
	return i18n.Tf(key, locale, field, e.Param())
}
