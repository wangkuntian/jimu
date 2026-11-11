// Package breach 检查口令是否出现在已知数据泄露集合中（Have I Been Pwned）。
//
// 采用 k-匿名范围查询：只把口令 SHA-1 的前 5 位十六进制字符发给服务端，完整口令与完整
// 哈希都不出网；响应携带同前缀的全部后缀哈希，本地比对是否命中。
package breach

import (
	"bufio"
	"context"
	"crypto/sha1" //nolint:gosec // HIBP 范围查询协议规定使用 SHA-1，仅作口令指纹，不用于存储或签名
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"jimu/internal/platform/httpclient"
)

// rangeURL 官方范围查询地址，k-匿名协议的固定入口
const rangeURL = "https://api.pwnedpasswords.com/range/"

// userAgent 出站请求标识，便于对方限流与排障
const userAgent = "jimu-breach-check"

// prefixLen SHA-1 十六进制前缀长度（协议固定 5 位）
const prefixLen = 5

// Checker 泄露口令检查器
type Checker interface {
	// IsBreached 返回口令是否已泄露（命中任意一条泄露记录即为 true）。
	// 网络或解析失败返回 error，由调用方决定放行还是拒绝。
	IsBreached(ctx context.Context, password string) (bool, error)
}

type hibpChecker struct {
	client  *httpclient.Client
	baseURL string
}

// New 创建基于 HIBP 范围查询的检查器
func New(client *httpclient.Client) Checker {
	return &hibpChecker{client: client, baseURL: rangeURL}
}

func (c *hibpChecker) IsBreached(ctx context.Context, password string) (bool, error) {
	//nolint:gosec // 见包注释：SHA-1 仅用于匹配泄露库指纹
	sum := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))

	// 使用 noctx 要求的 context 构造请求；URL 只含 5 位前缀
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+hash[:prefixLen], nil)
	if err != nil {
		return false, fmt.Errorf("build breach check request: %w", err)
	}
	// Add-Padding 让响应长度固定，避免旁观者按响应体积推断前缀命中数量
	req.Header.Set("Add-Padding", "true")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(ctx, req)
	if err != nil {
		return false, fmt.Errorf("breach check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("breach check returned status %d", resp.StatusCode)
	}

	suffix := hash[prefixLen:]
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, ":", 2)
		// 填充行没有冒号，直接跳过
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(parts[0], suffix) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read breach check response: %w", err)
	}
	return false, nil
}
