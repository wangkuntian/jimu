package assembly

import (
	"testing"

	"jimu/internal/contract"

	"github.com/stretchr/testify/require"
)

// TestValidatePortFlowRejectsReadBeforeProvide 护栏必须观察到真实的 Port 读取：
// 消费方先读到尚无人提供的端口即报错并点名能力与端口；提供者排在前则通过。
// 这正是 full 清单里 wireUser 读 "access" 而 wireAccess 未 Provide 时的失败形态。
func TestValidatePortFlowRejectsReadBeforeProvide(t *testing.T) {
	provider := Capability{
		Descriptor: contract.Descriptor{Name: "access"},
		Wire: func(ctx *Context) (contract.Module, error) {
			require.NoError(t, ctx.Provide("access", "role-assigner"))
			return fakeModule{name: "access"}, nil
		},
	}
	consumer := Capability{
		Descriptor: contract.Descriptor{Name: "user"},
		Wire: func(ctx *Context) (contract.Module, error) {
			_ = ctx.Port("access")
			return fakeModule{name: "user"}, nil
		},
	}

	err := ValidatePortFlow(Assembly{Name: "test", Capabilities: []Capability{consumer, provider}})
	require.ErrorContains(t, err, `capability "user" reads port "access" before it is provided`)

	require.NoError(t, ValidatePortFlow(Assembly{Name: "test", Capabilities: []Capability{provider, consumer}}))
}
