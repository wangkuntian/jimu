package main

import (
	"context"
	"errors"
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
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/auth"
	"jimu/internal/kernel/http/middleware"
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

// errProvisioningRequiresPublicRegistration 开通式注册要求公开注册（组合根跨字段校验）
var errProvisioningRequiresPublicRegistration = errors.New("auth.provisioning.enabled requires auth.public_registration")

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
			{Descriptor: authmodule.Descriptor, Wire: wireAuth},
			{Descriptor: passkeymodule.Descriptor, Wire: wirePasskey},
			{Descriptor: auditmodule.Descriptor, Wire: wireAudit},
			{Descriptor: consolemodule.Descriptor, Wire: wireConsole},
			{Descriptor: oauthmodule.Descriptor, Wire: wireOAuth},
			{Descriptor: apikey.Descriptor, Wire: wireAPIKey},
			{Descriptor: dataops.Descriptor, Wire: wireDataops},
			{Descriptor: feature.Descriptor, Wire: feature.Wire},
			{Descriptor: uploadsec.Descriptor, Wire: uploadsec.Wire},
			{Descriptor: search.Descriptor, Wire: search.Wire},
			{Descriptor: retention.Descriptor, Wire: wireRetention},
			{Descriptor: apidocs.Descriptor, Wire: wireAPIDocs},
			{Descriptor: grpcpkg.Descriptor, Wire: wireGRPC},
			{Descriptor: ws.Descriptor, Wire: ws.Wire},
		},
	}
}

func wireAuth(ctx *assembly.Context) (contract.Module, error) {
	authCfg := authConfig(ctx)
	if err := validateAuthConfig(authCfg); err != nil {
		return nil, err
	}
	// captcha 是 auth 的可选依赖（不在 Requires 内）：能力未启用/未装配时端口取回
	// nil，登录/注册跳过验证码校验。
	captchaVerifier, _ := ctx.Port(captcha.PortName).(contract.CaptchaVerifier)
	mod := authmodule.New(ctx.DB(), ctx.Redis(), *authCfg,
		ctx.Config().HTTP.Mode == config.HTTPModeRelease,
		captchaVerifier,
		ctx.Port(outbox.PortName), ctx.Port(notification.PortName), ctx.Port(encryption.PortName),
		ctx.Port(tenantmodule.PortName), ctx.Port(mfamodule.PortName), ctx.Port(tenantmodule.ProvisionerPortName), ctx.Port(breach.PortName))
	if err := ctx.Provide(authmodule.PortName, mod.Finalizer()); err != nil {
		return nil, err
	}
	return mod, nil
}

func wirePasskey(ctx *assembly.Context) (contract.Module, error) {
	finalizer, _ := ctx.Port(authmodule.PortName).(contract.LoginFinalizer)
	users, _ := ctx.Port(user.UserinfoPortName).(contract.UserinfoSource)
	return passkeymodule.New(passkeymodule.Deps{
		DB:         ctx.DB(),
		Redis:      ctx.Redis(),
		AuthCfg:    *authConfig(ctx),
		Users:      users,
		Finalizer:  finalizer,
		FailClosed: ctx.Config().HTTP.Mode == config.HTTPModeRelease,
	}), nil
}

func wireAudit(ctx *assembly.Context) (contract.Module, error) {
	return auditmodule.New(ctx.DB(), configSection(ctx, auditmodule.ConfigKey, func() *auditmodule.Config {
		return &auditmodule.Config{}
	}), ctx.Logger()), nil
}

func wireConsole(ctx *assembly.Context) (contract.Module, error) {
	cfg := ctx.Config()
	authCfg := authConfig(ctx)
	return consolemodule.New(cfg.Version, cfg.Environment, ctx.Redis(), ctx.DB(),
		auth.NewWithRotation(authCfg.JWTSecret, authCfg.JWTPreviousSecret, authCfg.Issuer, authCfg.AccessExpireMin, authCfg.RefreshExpireDay),
		ctx.EventBus(), middleware.IPAllowlist(cfg.Security.AdminIPAllowlist)), nil
}

func wireOAuth(ctx *assembly.Context) (contract.Module, error) {
	return oauthmodule.New(ctx.DB(), ctx.Redis(),
		configSection(ctx, oauthmodule.ConfigKey, func() *oauthmodule.Config { return &oauthmodule.Config{} }),
		*authConfig(ctx), ctx.HTTPClient()), nil
}

func wireAPIKey(ctx *assembly.Context) (contract.Module, error) {
	return apikey.New(ctx.DB(), ctx.Port(tenantmodule.PortName)), nil
}

func wireDataops(ctx *assembly.Context) (contract.Module, error) {
	return dataops.New(ctx.DB()), nil
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

func wireAPIDocs(ctx *assembly.Context) (contract.Module, error) {
	return apidocs.NewModule(ctx.Config().HTTP.Mode != config.HTTPModeRelease), nil
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

// configSection 取能力配置段并解引用为值；段不存在（能力未启用）时用 zero 构造零值，
// 与旧装配惯例一致（未启用能力的配置段取零值即"不接线/默认行为"）。
func configSection[T any](ctx *assembly.Context, key string, zero func() *T) T {
	if cfg := assembly.MustSection[*T](ctx, key); cfg != nil {
		return *cfg
	}
	return *zero()
}

// authConfig 取 auth 段；auth 未启用时该段不加载，回退为零值（旧装配惯例）。
func authConfig(ctx *assembly.Context) *authmodule.Config {
	if cfg := assembly.MustSection[*authmodule.Config](ctx, authmodule.ConfigKey); cfg != nil {
		return cfg
	}
	return &authmodule.Config{}
}

// validateAuthConfig 组合根承担的 auth 段跨字段校验。
// 设计 §8 ¶2 不拆 auth 段，provisioning 与 public_registration 同段；但 provisioning
// 的语义归 tenant，P2.1 裁定把这条校验留在组合根（与 outbox.publisher 依赖 queue.type 同理）。
func validateAuthConfig(cfg *authmodule.Config) error {
	if cfg.Provisioning.Enabled && !cfg.PublicRegistration {
		return errProvisioningRequiresPublicRegistration
	}
	return nil
}
