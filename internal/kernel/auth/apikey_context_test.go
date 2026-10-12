package auth

import (
	"context"
	"testing"

	"jimu/internal/kernel/tenant"
)

// A2.2 已校验的 Key 经 ContextWithAPIKey 注入后能被原样读回（机制视图的往返）。
func TestA2_2_VerifiedKeyRoundTripsThroughContext(t *testing.T) {
	key := &APIKey{ID: 7, TenantID: 5, Name: "svc", KeyPrefix: "jimu_beef", Enabled: true}
	ctx := ContextWithAPIKey(context.Background(), key)

	got, ok := APIKeyFromContext(ctx)
	if !ok || got != key {
		t.Fatalf("APIKeyFromContext = %#v/%v, want 注入的同一个 *APIKey", got, ok)
	}
	if tid := tenant.FromContext(ctx); tid != 0 {
		t.Fatalf("tenant.FromContext = %d, want 0（本函数只存 Key，租户由中间件按 Key 归属注入）", tid)
	}
}

// A2.1 未校验/未注入的 Key 不构成已认证，也不产生任何租户归属。
func TestA2_1_UnverifiedKeyContributesNoTenant(t *testing.T) {
	base := context.Background()
	if _, ok := APIKeyFromContext(base); ok {
		t.Fatal("没有 Key 的 context 不得被判为已认证")
	}
	// 类型不符的值放在正确 key 下也不得被当成 Key
	if _, ok := APIKeyFromContext(context.WithValue(base, APIKeyContextKey{}, "not-a-key")); ok {
		t.Fatal("类型不符的 context 值不得被当作已认证 Key")
	}
	// 反向对照：显式注入后才算已认证，但租户仍不由本函数内联归属
	key := &APIKey{ID: 7, TenantID: 5, Enabled: true}
	if tid := tenant.FromContext(ContextWithAPIKey(base, key)); tid != 0 {
		t.Fatalf("tenant.FromContext = %d, want 0", tid)
	}
}
