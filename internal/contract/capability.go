package contract

import "io/fs"

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

// Permission 能力声明的权限点：种子阶段写入 permissions 表并授予超管角色。
type Permission struct {
	Name     string // 中文名称（permissions.name）
	Resource string // 资源路径（permissions.resource，支持 keyMatch 通配）
	Action   string // HTTP 动作（permissions.action）
}

// ConfigSpec 声明能力拥有的一个配置段。
//
// 设计 §6.1/§8：能力配置由能力自身声明（默认值与校验），装配时按启用集合并；
// 未启用的能力其配置段既不出现也不校验。Section 支持点分路径（如
// "auth.webauthn"），因此能力可拥有嵌套段而对外配置布局保持不变。
type ConfigSpec struct {
	// Section 是 app.yaml 中的配置段键（点分路径）。
	Section string
	// New 返回本段配置实例的指针；框架把该段反序列化合并进去，随后调用
	// config.SectionConfig 的 ApplyDefaults/Validate（prod 下追加 ValidateProd）。
	New func() any
}

// Descriptor 能力对外的静态描述：用于启用闭包校验与路由挂载决策。
// 静态声明（而非运行时推断）是"删除能力后仍能编译"的前提。
type Descriptor struct {
	Name     string     // 能力名，全仓唯一
	Requires []string   // 硬依赖：启用本能力必须同时启用这些能力
	Mount    MountPoint // 路由挂载方式；零值等价 MountProtected

	// SoftRequires 可选依赖：目标能力缺失时本能力降级运行（不自动补齐、不参与拓扑序）。
	// 取值必须是 catalog 能力名，且不得与 Requires 重叠或自引用。
	SoftRequires []string

	// Drivers 本能力支持的驱动包名（= internal/capabilities/<cap>/<driver> 的目录名）；
	// 空表示本能力没有驱动概念。形态清单用 assembly.Capability.Drivers 选中子集，
	// 驱动包在 init() 中向能力核心注册（设计 §3.7）。
	Drivers []string

	// Permissions 能力拥有的权限点；种子时由启用集聚合写入，未启用的能力不种。
	Permissions []Permission

	// Owns 本能力拥有的表名；这些表只能由本能力的迁移 CREATE。
	// 空表示本能力不拥有表。
	Owns []string

	// Migrations 能力自带迁移的嵌入文件系统（根下应有 mysql/ 与 postgres/ 子目录）；
	// nil 表示该能力无迁移。
	Migrations fs.FS

	// Configs 能力拥有的配置段；nil 表示该能力的配置由内核段承担或无配置。
	// 一个能力可拥有多段（如 queue 同时拥有 queue 与 scheduler）。
	Configs []ConfigSpec
}

// Normalized 返回归一化后的挂载点：空值与任何未识别的取值（如大小写笔误）
// 都按 MountProtected 处理，避免特权路由被裸挂到根路由。
func (d Descriptor) Normalized() MountPoint {
	switch d.Mount {
	case MountPublic, MountSelfManaged:
		return d.Mount
	default:
		return MountProtected
	}
}

// Describable 由能力实现以声明自身描述。
type Describable interface {
	Descriptor() Descriptor
}

// Describe 读取能力描述；未实现 Describable 或传入 nil 时回退为零值描述。
func Describe(m Module) Descriptor {
	if m == nil {
		return Descriptor{}
	}
	if d, ok := m.(Describable); ok {
		return d.Descriptor()
	}
	return Descriptor{Name: m.Name()}
}
