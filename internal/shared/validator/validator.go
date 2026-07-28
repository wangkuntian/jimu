package validator

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var v *validator.Validate

func init() {
	v, _ = binding.Validator.Engine().(*validator.Validate)
	if v != nil {
		// 注册自定义校验规则
		_ = v.RegisterValidation("mobile", validateMobile)
		_ = v.RegisterValidation("password", validatePassword)
		_ = v.RegisterValidation("idcard", validateIDCard)
		_ = v.RegisterValidation("username", validateUsername)
	}
}

// Validate 获取校验器实例
func Validate() *validator.Validate {
	return v
}

// validateMobile 校验中国大陆手机号
func validateMobile(fl validator.FieldLevel) bool {
	mobile := fl.Field().String()
	if mobile == "" {
		return true // 空值由 required 处理
	}
	pattern := `^1[3-9]\d{9}$`
	return regexp.MustCompile(pattern).MatchString(mobile)
}

// validatePassword 校验密码强度（8-32位，含字母+数字）
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if password == "" {
		return true
	}
	if len(password) < 8 || len(password) > 32 {
		return false
	}
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)
	return hasLetter && hasDigit
}

// validateIDCard 校验中国大陆身份证号
func validateIDCard(fl validator.FieldLevel) bool {
	idcard := fl.Field().String()
	if idcard == "" {
		return true
	}
	pattern := `^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`
	return regexp.MustCompile(pattern).MatchString(idcard)
}

// validateUsername 校验用户名（4-20位字母数字下划线）
func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	if username == "" {
		return true
	}
	pattern := `^[a-zA-Z0-9_]{4,20}$`
	return regexp.MustCompile(pattern).MatchString(username)
}
