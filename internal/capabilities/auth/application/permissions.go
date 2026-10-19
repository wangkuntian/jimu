package application

import (
	"sort"

	"jimu/internal/kernel/access"
)

// PermissionPoint 一个有效权限点：与 Casbin 策略同形（resource 支持 keyMatch 通配）。
// 客户端按 (resource, action) 判定，不需要后端再翻译成权限码。
type PermissionPoint struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// EffectivePermissions 由「用户角色」与「全量角色策略」推导有效权限点。
//
// 语义与 kernel/access 的判定保持一致：用户命中某角色即拥有该角色的全部策略点；
// 去重后按 (resource, action) 排序，保证同一输入永远得到同一顺序（便于断言与 diff）。
// 无有效权限时返回**空切片而非 nil**：响应的 permissions 必须是 [] 而不是 null。
func EffectivePermissions(roles []string, policies []access.Policy) []PermissionPoint {
	owned := make(map[string]bool, len(roles))
	for _, role := range roles {
		owned[role] = true
	}

	seen := make(map[PermissionPoint]bool, len(policies))
	out := make([]PermissionPoint, 0, len(policies))
	for _, policy := range policies {
		if !owned[policy.Role] {
			continue
		}
		point := PermissionPoint{Resource: policy.Resource, Action: policy.Action}
		if seen[point] {
			continue
		}
		seen[point] = true
		out = append(out, point)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Resource != out[j].Resource {
			return out[i].Resource < out[j].Resource
		}
		return out[i].Action < out[j].Action
	})
	return out
}
