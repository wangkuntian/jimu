package capability

import "jimu/internal/contract"

// 以下别名让生成器核心包只 import internal/capability，而不直接 import
// internal/contract；组合事实的类型仍由 contract 定义，避免出现第二份描述。
type (
	// Descriptor 是 contract.Descriptor 的别名。
	Descriptor = contract.Descriptor
	// MountPoint 是 contract.MountPoint 的别名。
	MountPoint = contract.MountPoint
	// RuntimeService 是 contract.RuntimeService 的别名。
	RuntimeService = contract.RuntimeService
	// ConfigSpec 是 contract.ConfigSpec 的别名。
	ConfigSpec = contract.ConfigSpec
	// ProtectedHTTPMiddlewareMode 是 contract.ProtectedHTTPMiddlewareMode 的别名。
	ProtectedHTTPMiddlewareMode = contract.ProtectedHTTPMiddlewareMode
)

const (
	// MountPublic 见 contract.MountPublic。
	MountPublic = contract.MountPublic
	// MountProtected 见 contract.MountProtected。
	MountProtected = contract.MountProtected
	// MountSelfManaged 见 contract.MountSelfManaged。
	MountSelfManaged = contract.MountSelfManaged

	// ProtectedHTTPMiddlewareNone 见 contract.ProtectedHTTPMiddlewareNone。
	ProtectedHTTPMiddlewareNone = contract.ProtectedHTTPMiddlewareNone
	// ProtectedHTTPMiddlewarePrimary 见 contract.ProtectedHTTPMiddlewarePrimary。
	ProtectedHTTPMiddlewarePrimary = contract.ProtectedHTTPMiddlewarePrimary
	// ProtectedHTTPMiddlewareFallback 见 contract.ProtectedHTTPMiddlewareFallback。
	ProtectedHTTPMiddlewareFallback = contract.ProtectedHTTPMiddlewareFallback
)
