package interfaces

// Config 认证路由与处理器所需的配置视图。
//
// 完整配置（含 JWT/WebAuthn/Provisioning）由能力根包持有并声明为配置段；
// interfaces 是叶子包（根包 import 本包），不能反向 import 根包，故此处只保留
// 路由/处理器实际用到的字段，由根包在 RegisterHTTP 装配期映射。
type Config struct {
	PublicRegistration    bool // 是否注册公开注册路由
	ProvisioningEnabled   bool // 开通式注册（注册 = 开通新租户）
	LoginRateLimit        int
	LoginRateWindowSec    int
	RegisterRateLimit     int
	RegisterRateWindowSec int
}
