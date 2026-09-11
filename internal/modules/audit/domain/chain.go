package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash"
	"time"
)

// separator 分隔 prevHash 与规范化载荷，避免拼接歧义
const separator = "\x1f"

// chainPayload 链式哈希的规范化载荷：字段与顺序固定，保证跨进程与跨版本可复现。
// 刻意不含 ID（雪花 ID 在插入时才由 hook 生成）与 GORM 自动维护的字段。
type chainPayload struct {
	TenantID  uint64 `json:"tenant_id"`
	UserID    uint64 `json:"user_id"`
	Username  string `json:"username"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Detail    string `json:"detail"`
	Changes   string `json:"changes"`
	IP        string `json:"ip"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"created_at_unix"` // 秒级，与数据库 TIMESTAMP 精度一致
}

// NormalizeCreatedAt 归一化写入时间到秒级 UTC：
// 数据库 TIMESTAMP 精度为秒，若哈希包含毫秒则校验时必然不一致。
func NormalizeCreatedAt(t time.Time) time.Time {
	return t.UTC().Truncate(time.Second)
}

// ComputeEntryHash 计算本条审计的链式哈希：
//
//	secret 非空：HMAC-SHA256(secret, prevHash + payload)
//	secret 为空：SHA-256(prevHash + payload)
//
// 前者可防止持有库写权限者在篡改后重算整条链；后者只能发现就地篡改，
// 因此生产环境建议配置 audit.hash_secret。
func (l *AuditLog) ComputeEntryHash(secret []byte) string {
	payload, err := json.Marshal(chainPayload{
		TenantID:  l.TenantID,
		UserID:    l.UserID,
		Username:  l.Username,
		Action:    l.Action,
		Resource:  l.Resource,
		Detail:    l.Detail,
		Changes:   l.ChangesRaw,
		IP:        l.IP,
		Method:    l.Method,
		Path:      l.Path,
		Status:    l.Status,
		CreatedAt: NormalizeCreatedAt(l.CreatedAt).Unix(),
	})
	if err != nil {
		// json.Marshal 对纯字符串/数值结构不会失败，保守返回空哈希以拒绝写入
		return ""
	}

	var mac hash.Hash
	if len(secret) > 0 {
		mac = hmac.New(sha256.New, secret)
	} else {
		mac = sha256.New()
	}
	mac.Write([]byte(l.PrevHash))
	mac.Write([]byte(separator))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// MatchesEntryHash 校验本条记录内容与存储的 entry_hash 是否一致
func (l *AuditLog) MatchesEntryHash(secret []byte) bool {
	if l.EntryHash == "" {
		return false
	}
	return hmac.Equal([]byte(l.ComputeEntryHash(secret)), []byte(l.EntryHash))
}
