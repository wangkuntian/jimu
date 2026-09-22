package main

import (
	"context"
	"fmt"
	"os"
	"time"

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
	"jimu/internal/kernel/scheduler"
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

// fullAssembly 是本阶段的过渡形态：能力清单 + 逐能力的内联 Wire。Task 3 会把每个
// 内联闭包搬到 internal/capabilities/<name>/wire.go，此处只剩清单。
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
			{Descriptor: retention.Descriptor, Wire: wireRetention},
			{Descriptor: apidocs.Descriptor, Wire: apidocs.Wire},
			{Descriptor: grpcpkg.Descriptor, Wire: wireGRPC},
			{Descriptor: ws.Descriptor, Wire: ws.Wire},
		},
	}
}

func wireAPIKey(ctx *assembly.Context) (contract.Module, error) {
	return apikey.New(ctx.DB(), ctx.Port(tenantmodule.PortName)), nil
}

func wireRetention(ctx *assembly.Context) (contract.Module, error) {
	cfg, err := retention.Load(ctx.Sections())
	if err != nil {
		return nil, fmt.Errorf("init retention config: %w", err)
	}
	db := ctx.DB()
	if db != nil {
		cleanupSvc := retention.NewCleanupService(db, retention.DefaultCleanupConfig())
		if err := ctx.RegisterJob(scheduler.Job{ID: "cleanup", Name: "Data Cleanup", Spec: "0 3 * * *", Run: func() {
			results, err := cleanupSvc.Run(context.Background())
			if err != nil {
				ctx.Logger().Errorw("cleanup job failed", "error", err.Error())
				return
			}
			for _, r := range results {
				if r.Deleted > 0 {
					ctx.Logger().Infow("cleanup completed", "table", r.Table, "deleted", r.Deleted)
				}
			}
		}}); err != nil {
			return nil, err
		}
	}
	if db != nil && cfg.Enabled {
		retentionSvc := retention.NewRetentionService(db, *cfg)
		spec := cfg.Cron
		if spec == "" {
			spec = "30 3 * * *"
		}
		if err := ctx.RegisterJob(scheduler.Job{ID: "retention", Name: "History Retention", Spec: spec, Run: func() {
			runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			results, err := retentionSvc.Run(runCtx)
			if err != nil {
				ctx.Logger().Errorw("retention job failed", "error", err.Error())
				return
			}
			for _, r := range results {
				if r.Deleted > 0 {
					ctx.Logger().Infow("retention completed", "table", r.Table, "deleted", r.Deleted)
				}
			}
		}}); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func wireGRPC(ctx *assembly.Context) (contract.Module, error) {
	cfg := ctx.Config()
	// gRPC server（与 HTTP 双栈；enabled 时纳入生命周期）
	grpcServer, err := grpcpkg.New(grpcpkg.Config{
		Enabled:    cfg.GRPC.Enabled,
		Host:       cfg.GRPC.Host,
		Port:       cfg.GRPC.Port,
		TimeoutSec: cfg.GRPC.TimeoutSec,
		TLS:        cfg.GRPC.TLS,
	}, ctx.Logger(), ctx.Reporter())
	if err != nil {
		return nil, fmt.Errorf("init grpc server: %w", err)
	}
	// 业务示例：注册 UserInfoService，用户数据经 contract.UserinfoSource 端口读取
	// （user 能力提供适配实现，grpc 能力不直接依赖 user/domain）
	if source, ok := ctx.Port(user.UserinfoPortName).(contract.UserinfoSource); ok && source != nil {
		grpcServer.RegisterUserInfoService(source)
	}
	if cfg.GRPC.Enabled {
		ctx.RegisterComponent(grpcServer)
	}
	return nil, nil
}
