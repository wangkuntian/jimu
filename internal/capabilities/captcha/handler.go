package captcha

import (
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// CaptchaHandler 验证码 HTTP 处理器
type CaptchaHandler struct {
	service *Service
}

// NewCaptchaHandler 创建验证码处理器
func NewCaptchaHandler(service *Service) *CaptchaHandler {
	return &CaptchaHandler{service: service}
}

// Generate godoc
// @Summary      获取验证码
// @Description  生成图形验证码，返回 captcha_id 与 base64 图片
// @Tags         验证码
// @Produce      json
// @Success      200  {object}  response.Body  "成功，返回 captcha_id 和 captcha_image"
// @Router       /captcha [get]
func (h *CaptchaHandler) Generate(c *gin.Context) {
	id, img, err := h.service.Generate(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"captcha_id":    id,
		"captcha_image": img,
	})
}
