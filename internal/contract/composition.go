package contract

// Validity 描述一个能力组合的结构合法性，与支持级别正交。
type Validity string

const (
	// ValidityValid 组合通过了全部静态组合规则。
	ValidityValid Validity = "valid"
	// ValidityInvalid 组合违反至少一条静态组合规则。
	ValidityInvalid Validity = "invalid"
)

// Valid 判断取值是否为已定义的合法性状态。
func (v Validity) Valid() bool {
	return v == ValidityValid || v == ValidityInvalid
}

// SupportLevel 描述一个能力组合获得的官方支持程度，与合法性正交：
// 非法组合仍可带支持级别用于报告，合法组合也未必达到 certified。
type SupportLevel string

const (
	// SupportCertified 官方模板预设组合（basic-api / auth-api）。
	SupportCertified SupportLevel = "certified"
	// SupportValidated 通过全部门禁的固定 catalog 子集。
	SupportValidated SupportLevel = "validated"
	// SupportExperimental 专家组合或未经完整验证的子集。
	SupportExperimental SupportLevel = "experimental"
)

// Valid 判断取值是否为已定义的支持级别。
func (s SupportLevel) Valid() bool {
	switch s {
	case SupportCertified, SupportValidated, SupportExperimental:
		return true
	default:
		return false
	}
}

// ProvidedPort 声明能力在装配期注册的 contract 端口。
// Name 必须与 assembly.Context.Provide 使用的名字一致。
type ProvidedPort struct {
	Name string
}

// ConsumedPort 声明能力从装配上下文读取的 contract 端口。
type ConsumedPort struct {
	Name string
	// Required 为 true 表示该端口是硬前提：组合中必须存在提供方。
	// 为 false 时缺失由能力自行降级（对应 SoftRequires 的语义）。
	Required bool
}

// DriverSpec 为一个已声明的驱动补充组合元数据。
// Name 必须命中 Descriptor.Drivers；未列出的驱动视为没有额外要求。
type DriverSpec struct {
	Name string
	// RuntimeServices 是选中该驱动后额外需要的外部服务，例如 queue/redis 需要 redis。
	RuntimeServices []RuntimeService
}
