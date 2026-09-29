package mfa

// Config mfa 能力的装配期配置（不来自 YAML 段）。
//
// mfa 的 Requires 只有 user（不依赖 auth），故**不得** import auth 能力类型；
// 本结构由组合根从 auth 段取值后传入（JWT 参数用于保护本能力的路由，
// TrustedDeviceDays/Issuer 用于可信设备与 TOTP 开户 URI）。
type Config struct {
	JWTSecret         string
	JWTPreviousSecret string
	Issuer            string
	AccessExpireMin   int
	RefreshExpireDay  int
	TrustedDeviceDays int
}
