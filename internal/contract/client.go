package contract

import "context"

// ClientInfo 登录请求客户端信息，供 auth / mfa / passkey 共享。
// IP/UserAgent 用于登录历史；DeviceToken/RememberDevice 用于可信设备（跳过 TOTP）。
type ClientInfo struct {
	IP             string
	UserAgent      string
	DeviceToken    string
	RememberDevice bool
}

type clientInfoKey struct{}

// WithClientInfo 把客户端信息写入上下文（保留已设置的其他字段）。
func WithClientInfo(ctx context.Context, ip, userAgent string) context.Context {
	info := ClientInfoFrom(ctx)
	info.IP = ip
	info.UserAgent = userAgent
	return context.WithValue(ctx, clientInfoKey{}, info)
}

// WithLoginDevice 写入可信设备相关参数（保留已设置的客户端信息）。
func WithLoginDevice(ctx context.Context, deviceToken string, remember bool) context.Context {
	info := ClientInfoFrom(ctx)
	info.DeviceToken = deviceToken
	info.RememberDevice = remember
	return context.WithValue(ctx, clientInfoKey{}, info)
}

// ClientInfoFrom 读取客户端信息；未设置时返回零值。
func ClientInfoFrom(ctx context.Context) ClientInfo {
	if v, ok := ctx.Value(clientInfoKey{}).(ClientInfo); ok {
		return v
	}
	return ClientInfo{}
}
