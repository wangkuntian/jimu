package main

import (
	"fmt"
	"os"

	"jimu/internal/assembly"
	accessmodule "jimu/internal/capabilities/access"
	"jimu/internal/capabilities/apidocs"
	"jimu/internal/capabilities/apikey"
	auditmodule "jimu/internal/capabilities/audit"
	authmodule "jimu/internal/capabilities/auth"
	"jimu/internal/capabilities/breach"
	"jimu/internal/capabilities/captcha"
	consolemodule "jimu/internal/capabilities/console"
	"jimu/internal/capabilities/dataops"
	"jimu/internal/capabilities/encryption"
	"jimu/internal/capabilities/feature"
	grpcpkg "jimu/internal/capabilities/grpc"
	mfamodule "jimu/internal/capabilities/mfa"
	"jimu/internal/capabilities/notification"
	oauthmodule "jimu/internal/capabilities/oauth"
	"jimu/internal/capabilities/outbox"
	passkeymodule "jimu/internal/capabilities/passkey"
	"jimu/internal/capabilities/queue"
	"jimu/internal/capabilities/retention"
	"jimu/internal/capabilities/search"
	"jimu/internal/capabilities/storage"
	tenantmodule "jimu/internal/capabilities/tenant"
	"jimu/internal/capabilities/uploadsec"
	"jimu/internal/capabilities/user"
	"jimu/internal/capabilities/ws"
	"jimu/internal/contract"
)

// @title           Jimu API
// @version         1.0
// @description     Jimu 后端框架 API - 提供用户认证、权限管理、角色管理、系统监控等功能
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization

// version 版本号，通过 ldflags 注入：-ldflags "-X main.version=v0.1.0"
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	return assembly.Run(fullAssembly())
}

// fullAssembly 是本阶段的过渡形态：能力清单（每项引用能力自导出的 Descriptor 与
// Wire），能力件一律由 capabilities/<name>/wire.go 自装配，组合根不再构造任何能力件。
//
// 顺序即装配顺序：提供端口的能力必须排在消费它的能力之前（encryption/storage/
// notification/queue/outbox/breach 先于 tenant/user/auth/uploadsec/grpc；
// tenant/access 先于 user；captcha/mfa 先于 auth）。
// 无 Module 实例的能力（outbox/search/breach/ws 等）保留条目并给出空 Wire，
// 使启用集与 /capabilities 报告逐值一致。
func fullAssembly() assembly.Assembly {
	return assembly.Assembly{
		Name:    "full",
		Version: version,
		Capabilities: []assembly.Capability{
			{Descriptor: encryption.Descriptor, Wire: encryption.Wire},
			{Descriptor: storage.Descriptor, Wire: storage.Wire},
			{Descriptor: notification.Descriptor, Wire: notification.Wire},
			{Descriptor: queue.Descriptor, Wire: queue.Wire},
			{Descriptor: outbox.Descriptor, Wire: outbox.Wire},
			{Descriptor: breach.Descriptor, Wire: breach.Wire},
			{Descriptor: tenantmodule.Descriptor, Wire: tenantmodule.Wire},
			{Descriptor: accessmodule.Descriptor, Wire: accessmodule.Wire},
			{Descriptor: user.Descriptor, Wire: user.Wire},
			{Descriptor: captcha.Descriptor, Wire: captcha.Wire}, // 已搬迁的试点能力
			{Descriptor: mfamodule.Descriptor, Wire: mfamodule.Wire},
			{Descriptor: authmodule.Descriptor, Wire: authmodule.Wire},
			{Descriptor: passkeymodule.Descriptor, Wire: passkeymodule.Wire},
			{Descriptor: auditmodule.Descriptor, Wire: auditmodule.Wire},
			{Descriptor: consolemodule.Descriptor, Wire: consolemodule.Wire},
			{Descriptor: oauthmodule.Descriptor, Wire: oauthmodule.Wire},
			{Descriptor: apikey.Descriptor, Wire: wireAPIKey},
			{Descriptor: dataops.Descriptor, Wire: dataops.Wire},
			{Descriptor: feature.Descriptor, Wire: feature.Wire},
			{Descriptor: uploadsec.Descriptor, Wire: uploadsec.Wire},
			{Descriptor: search.Descriptor, Wire: search.Wire},
			{Descriptor: retention.Descriptor, Wire: retention.Wire},
			{Descriptor: apidocs.Descriptor, Wire: apidocs.Wire},
			{Descriptor: grpcpkg.Descriptor, Wire: grpcpkg.Wire},
			{Descriptor: ws.Descriptor, Wire: ws.Wire},
		},
	}
}

func wireAPIKey(ctx *assembly.Context) (contract.Module, error) {
	return apikey.New(ctx.DB(), ctx.Port(tenantmodule.PortName)), nil
}
