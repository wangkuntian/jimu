package contract

// MountPoint 决定能力的 HTTP 路由挂载方式。
type MountPoint string

const (
	// MountPublic 挂在 /api/v1 上且不套用受保护中间件（登录、回调、验证码等公开端点）。
	MountPublic MountPoint = "public"
	// MountProtected 挂在 /api/v1 上并套用受保护中间件（认证 + 租户注入 + 限流）。
	MountProtected MountPoint = "protected"
	// MountSelfManaged 由能力自行决定挂载与中间件（如 auth 内部同时有公开与受保护子组）。
	MountSelfManaged MountPoint = "self-managed"
)

// Descriptor 能力对外的静态描述：用于启用闭包校验与路由挂载决策。
// 静态声明（而非运行时推断）是"删除能力后仍能编译"的前提。
type Descriptor struct {
	Name     string     // 能力名，全仓唯一
	Requires []string   // 硬依赖：启用本能力必须同时启用这些能力
	Mount    MountPoint // 路由挂载方式；零值等价 MountProtected
}

// Normalized 返回归一化后的挂载点，空值按 MountProtected 处理。
func (d Descriptor) Normalized() MountPoint {
	if d.Mount == "" {
		return MountProtected
	}
	return d.Mount
}

// Describable 由能力实现以声明自身描述。
type Describable interface {
	Descriptor() Descriptor
}

// Describe 读取能力描述；未实现 Describable 时回退为"仅名称 + 受保护挂载"。
func Describe(m Module) Descriptor {
	if d, ok := m.(Describable); ok {
		return d.Descriptor()
	}
	return Descriptor{Name: m.Name()}
}
