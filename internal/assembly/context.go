package assembly

import (
	"fmt"

	"jimu/internal/app"
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/kernel/event"
	"jimu/internal/kernel/httpclient"
	"jimu/internal/kernel/logger"
	redistore "jimu/internal/kernel/redis"
	"jimu/internal/kernel/scheduler"

	"gorm.io/gorm"
)

// Context 是能力 Wire 的装配上下文：暴露内核件访问器，并持有模块/端口注册表。
// 由 Run 构造后传给每个能力的 Wire；能力只经它取得依赖 —— 不 import 其它能力包，
// 也不 import internal/app。
type Context struct {
	container   *app.Container
	sections    config.SectionDecoder
	capCfgs     *app.CapabilityConfigs
	ports       map[string]any
	modules     []contract.Module
	moduleNames map[string]bool
	// onPort 在每次 Port 读取时回调（ValidatePortFlow 用它观察读取结果；生产路径为 nil）。
	// provided 表示该端口在读取发生时已注册。
	onPort func(name string, provided bool)
}

// newContext 由内核容器与已解码的配置段构造装配上下文。
func newContext(container *app.Container, sections config.SectionDecoder, capCfgs *app.CapabilityConfigs) *Context {
	return &Context{
		container:   container,
		sections:    sections,
		capCfgs:     capCfgs,
		ports:       make(map[string]any),
		moduleNames: make(map[string]bool),
	}
}

// Config 全量配置。
func (c *Context) Config() *config.Config { return c.container.Config }

// Logger 内核日志器。
func (c *Context) Logger() *logger.Logger { return c.container.Logger }

// DB gorm 句柄。
func (c *Context) DB() *gorm.DB { return c.container.DB }

// Redis 缓存客户端。
func (c *Context) Redis() redistore.Client { return c.container.Redis }

// Sections 按 YAML 点分键解码配置段。
func (c *Context) Sections() config.SectionDecoder { return c.sections }

// CapabilityConfigs 已按启用集解码并校验的能力配置段。
func (c *Context) CapabilityConfigs() *app.CapabilityConfigs { return c.capCfgs }

// EventBus 全局事件总线。
func (c *Context) EventBus() *event.EventBus { return c.container.EventBus }

// HTTPClient 统一出站 HTTP client。
func (c *Context) HTTPClient() *httpclient.Client { return c.container.HTTPClient }

// Scheduler 定时任务调度器。
func (c *Context) Scheduler() *scheduler.CronScheduler { return c.container.Scheduler }

// Provide 注册一个端口实现，供排在后面的能力经 Port 消费。
// 同名端口重复注册报错：静默覆盖会让装配顺序的语义变得不可推断。
func (c *Context) Provide(name string, port any) error {
	if name == "" {
		return fmt.Errorf("assembly: port name must not be empty")
	}
	if _, dup := c.ports[name]; dup {
		return fmt.Errorf("assembly: port %q provided twice", name)
	}
	c.ports[name] = port
	return nil
}

// Port 取回端口实现；未提供时返回 nil（消费方按软依赖降级，例如跳过验证码校验）。
func (c *Context) Port(name string) any {
	v, ok := c.ports[name]
	if c.onPort != nil {
		c.onPort(name, ok)
	}
	return v
}

// Register 登记能力的 Module 实例（按调用顺序），重名报错。
// Wire 返回 nil 的能力（无 Module 实例、只提供端口或仅参与迁移）不需要注册。
func (c *Context) Register(module contract.Module) error {
	name := contract.Describe(module).Name
	if name == "" {
		return fmt.Errorf("assembly: module has an empty capability name")
	}
	if c.moduleNames[name] {
		return fmt.Errorf("assembly: capability %q registered twice", name)
	}
	c.moduleNames[name] = true
	c.modules = append(c.modules, module)
	return nil
}

// MustSection 取某能力配置段并断言为具体类型 T（T 为指针类型，如 *captcha.Config）；
// 段不存在（能力未启用或未声明该段）时返回 T 的零值。这是给能力 Wire 的便利入口：
// 取配置无需手写类型断言，也不必 import internal/app。
func MustSection[T any](ctx *Context, key string) T {
	v, _ := app.SectionOf[T](ctx.CapabilityConfigs(), key)
	return v
}
