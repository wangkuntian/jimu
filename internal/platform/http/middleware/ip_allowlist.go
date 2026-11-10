package middleware

import (
	"net"
	"strings"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// IPAllowlist 限制来源 IP（CIDR 或单个 IP）：
//   - 列表为空：不限制（放行所有请求）
//   - 列表非空：仅放行命中条目的来源，其余返回 403
//
// 客户端 IP 由 gin 依据 http.trusted_proxies 解析（含 X-Forwarded-For），
// 因此必须正确配置可信代理，否则白名单可被伪造头绕过。
//
// 条目在 config.Validate 阶段已校验，这里对仍无法解析的条目按忽略处理。
func IPAllowlist(cidrs []string) gin.HandlerFunc {
	// 空列表 = 未启用白名单
	if len(cidrs) == 0 {
		return func(c *gin.Context) { c.Next() }
	}

	nets := parseCIDRs(cidrs)
	// 列表非空但无有效条目：fail-closed（config.Validate 已在启动阶段拦截该情况）
	if len(nets) == 0 {
		return func(c *gin.Context) {
			response.Fail(c, errors.New(errors.CodeForbidden, "ip not allowed"))
			c.Abort()
		}
	}

	return func(c *gin.Context) {
		ip := net.ParseIP(c.ClientIP())
		if ip == nil || !containsIP(nets, ip) {
			response.Fail(c, errors.New(errors.CodeForbidden, "ip not allowed"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// parseCIDRs 解析白名单条目；单个 IP 视为 /32（IPv4）或 /128（IPv6）
func parseCIDRs(entries []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, block, err := net.ParseCIDR(entry); err == nil {
			nets = append(nets, block)
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}
	return nets
}

func containsIP(nets []*net.IPNet, ip net.IP) bool {
	for _, block := range nets {
		// 兼容 IPv4-mapped IPv6（::ffff:1.2.3.4）
		if block.Contains(ip) {
			return true
		}
		if v4 := ip.To4(); v4 != nil && block.Contains(v4) {
			return true
		}
	}
	return false
}
