package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"jimu/internal/platform/observability"
	"jimu/internal/platform/reporter"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// 枚举常量
const (
	HTTPModeDebug   = "debug"
	HTTPModeRelease = "release"
	HTTPModeTest    = "test"

	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"

	LogFormatJSON    = "json"
	LogFormatConsole = "console"

	QueueTypeRedis    = "redis"
	QueueTypeKafka    = "kafka"
	QueueTypeRabbitMQ = "rabbitmq"

	OutboxPublisherEventBus = "event_bus"
	OutboxPublisherMQ       = "mq"

	SchedulerStoreMemory = "memory"
	SchedulerStoreMySQL  = "mysql"

	DBDriverMySQL    = "mysql"
	DBDriverPostgres = "postgres"
	DBDriverMariaDB  = "mariadb"

	RedisModeSingle   = "single"   // 单机模式（默认）
	RedisModeSentinel = "sentinel" // 哨兵模式（高可用）
	RedisModeCluster  = "cluster"  // 集群模式
)

var (
	validHTTPModes          = []string{HTTPModeDebug, HTTPModeRelease, HTTPModeTest}
	validLogLevels          = []string{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}
	validLogFormats         = []string{LogFormatJSON, LogFormatConsole}
	validQueueTypes         = []string{QueueTypeRedis, QueueTypeKafka, QueueTypeRabbitMQ}
	validOutboxPublishers   = []string{OutboxPublisherEventBus, OutboxPublisherMQ}
	validOutboxMQQueueTypes = []string{QueueTypeKafka, QueueTypeRabbitMQ, QueueTypeRedis}
	validSchedulerStores    = []string{SchedulerStoreMemory, SchedulerStoreMySQL}
	validDBDrivers          = []string{DBDriverMySQL, DBDriverPostgres, DBDriverMariaDB, ""}
	validRedisModes         = []string{RedisModeSingle, RedisModeSentinel, RedisModeCluster}
)

// QueueConfig 队列配置
type QueueConfig struct {
	Type     string              `mapstructure:"type"`     // 队列类型：redis, kafka, rabbitmq
	Kafka    QueueKafkaConfig    `mapstructure:"kafka"`    // Kafka 队列配置（type=kafka 时使用）
	RabbitMQ QueueRabbitMQConfig `mapstructure:"rabbitmq"` // RabbitMQ 队列配置（type=rabbitmq 时使用）
}

// OutboxConfig Outbox 配置
type OutboxConfig struct {
	Publisher string `mapstructure:"publisher"` // 发布器类型：event_bus, mq
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	Store string `mapstructure:"store"` // 任务定义存储类型：memory, mysql
}

// OAuthProviderConfig 单个 OAuth 提供商配置
type OAuthProviderConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	Enabled      bool   `mapstructure:"enabled"`
}

// OAuthConfig OAuth 登录配置
type OAuthConfig struct {
	Providers map[string]OAuthProviderConfig `mapstructure:"providers"` // 提供商名 -> 配置（google/github/wechat）
}

// CaptchaConfig 验证码配置
type CaptchaConfig struct {
	Enabled bool `mapstructure:"enabled"` // 是否启用登录/注册验证码
	TTLMin  int  `mapstructure:"ttl_min"` // 验证码有效期（分钟）
}

// EmailConfig 邮件通知配置
type EmailConfig struct {
	Enabled  bool   `mapstructure:"enabled"`  // 是否启用真实 SMTP 发送；false 时回退日志渠道
	Host     string `mapstructure:"host"`     // SMTP 服务器地址
	Port     int    `mapstructure:"port"`     // SMTP 端口（通常 25/465/587）
	Username string `mapstructure:"username"` // 认证用户名
	Password string `mapstructure:"password"` // 认证密码（敏感，建议环境变量注入）
	From     string `mapstructure:"from"`     // 发件人地址
}

// SMSConfig 短信通知配置
type SMSConfig struct {
	Enabled   bool   `mapstructure:"enabled"`    // 是否启用真实短信发送；false 时回退日志渠道
	Provider  string `mapstructure:"provider"`   // 短信服务商：aliyun
	APIKey    string `mapstructure:"api_key"`    // AccessKey ID（敏感，建议环境变量注入）
	APISecret string `mapstructure:"api_secret"` // AccessKey Secret（敏感，建议环境变量注入）
	SignName  string `mapstructure:"sign_name"`  // 短信签名
}

// CaptchaResult 验证码返回
type CaptchaResult struct {
	CaptchaID    string `json:"captcha_id"`
	CaptchaImage string `json:"captcha_image"`
}

// QueueKafkaConfig Kafka 队列配置
type QueueKafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`  // broker 地址列表，如 ["kafka:9092"]
	Topic   string   `mapstructure:"topic"`    // 消费/生产主题
	GroupID string   `mapstructure:"group_id"` // 消费组 ID
}

// QueueRabbitMQConfig RabbitMQ 队列配置
type QueueRabbitMQConfig struct {
	URL      string `mapstructure:"url"`      // AMQP URL，如 amqp://guest:guest@rabbitmq:5672/
	Queue    string `mapstructure:"queue"`    // 队列名
	Exchange string `mapstructure:"exchange"` // 交换机名（留空则使用默认直连交换机）
}

// IDConfig 雪花 ID 配置
type IDConfig struct {
	WorkerID int64 `mapstructure:"worker_id"` // worker 编号（0-1023），多实例部署时每个副本需唯一
}

type Config struct {
	HTTP         HTTPConfig                  `mapstructure:"http"`
	Management   ManagementConfig            `mapstructure:"management"`
	DB           DBConfig                    `mapstructure:"db"`
	Redis        RedisConfig                 `mapstructure:"redis"`
	Log          LogConfig                   `mapstructure:"log"`
	Auth         AuthConfig                  `mapstructure:"auth"`
	Server       ServerConfig                `mapstructure:"server"`
	ID           IDConfig                    `mapstructure:"id"`
	Cache        CacheConfig                 `mapstructure:"cache"`
	Audit        AuditConfig                 `mapstructure:"audit"`
	Storage      StorageConfig               `mapstructure:"storage"`
	Upload       UploadConfig                `mapstructure:"upload"`
	Security     SecurityConfig              `mapstructure:"security"`
	Queue        QueueConfig                 `mapstructure:"queue"`
	Outbox       OutboxConfig                `mapstructure:"outbox"`
	Scheduler    SchedulerConfig             `mapstructure:"scheduler"`
	OAuth        OAuthConfig                 `mapstructure:"oauth"`
	Captcha      CaptchaConfig               `mapstructure:"captcha"`
	Email        EmailConfig                 `mapstructure:"email"`
	SMS          SMSConfig                   `mapstructure:"sms"`
	Notification NotificationConfig          `mapstructure:"notification"`
	OTEL         observability.TracingConfig `mapstructure:"otel"`
	ErrorReport  reporter.ReporterConfig     `mapstructure:"error_reporting"`
	HTTPClient   HTTPClientConfig            `mapstructure:"http_client"`
	GRPC         GRPCConfig                  `mapstructure:"grpc"`
	// 元数据（非 YAML 配置，运行时注入）
	Version     string `mapstructure:"-"`
	Environment string `mapstructure:"-"`
}

// HTTPClientConfig 出站 HTTP client 配置
type HTTPClientConfig struct {
	TimeoutSec      int `mapstructure:"timeout_sec"`       // 单次请求超时（秒），0 用默认 10
	MaxRetries      int `mapstructure:"max_retries"`       // 失败重试次数，0 用默认 2
	RetryIntervalMS int `mapstructure:"retry_interval_ms"` // 重试基础间隔（毫秒），0 用默认 200
	RateLimitRate   int `mapstructure:"rate_limit_rate"`   // 每秒请求数（按目标 host 独立限流），0 不限流
	RateLimitBurst  int `mapstructure:"rate_limit_burst"`  // 令牌桶容量，0 用 rate（桶=平均速率）
}

// NotificationConfig 通知渠道配置
type NotificationConfig struct {
	Webhook WebhookNotificationConfig `mapstructure:"webhook"` // Webhook 渠道配置
}

// WebhookNotificationConfig Webhook 通知配置
type WebhookNotificationConfig struct {
	SignSecret string `mapstructure:"sign_secret"` // 载荷签名密钥（HMAC-SHA256）；空则不签名
}

// GRPCConfig gRPC server 配置（与 HTTP 双栈并存，可选启用）
type GRPCConfig struct {
	Enabled bool   `mapstructure:"enabled"` // 是否启用 gRPC server
	Host    string `mapstructure:"host"`    // 监听地址
	Port    int    `mapstructure:"port"`    // 监听端口
}

// ServerConfig 服务运行时配置
type ServerConfig struct {
	TimeoutSec     int `mapstructure:"timeout_sec"`      // 请求超时秒数，0 表示不限制
	RateLimitRate  int `mapstructure:"rate_limit_rate"`  // 全局限流速率（每秒请求数），0 表示不限流
	RateLimitBurst int `mapstructure:"rate_limit_burst"` // 限流桶容量，允许的突发请求数
}

// SecurityConfig 安全头配置
type SecurityConfig struct {
	ContentTypeOptions    string `mapstructure:"content_type_options"`    // X-Content-Type-Options
	FrameOptions          string `mapstructure:"frame_options"`           // X-Frame-Options
	XSSProtection         string `mapstructure:"xss_protection"`          // X-XSS-Protection
	StrictTransport       string `mapstructure:"strict_transport"`        // Strict-Transport-Security
	ContentSecurityPolicy string `mapstructure:"content_security_policy"` // Content-Security-Policy
	ReferrerPolicy        string `mapstructure:"referrer_policy"`         // Referrer-Policy
	PermissionsPolicy     string `mapstructure:"permissions_policy"`      // Permissions-Policy

	// CSRF 防护密钥。非空时启用 CSRF 中间件（Bearer 认证请求自动跳过，不影响 JWT API）
	CSRFSecret string `mapstructure:"csrf_secret"`

	// 字段级加密密钥（AES-256-GCM，≥32 字节）。空则明文模式（email/phone 不加密存储，仍计算盲索引）。
	// 建议 ENCRYPTION_KEY 环境变量注入；启用后存量明文行可正常解密读回。
	EncryptionKey string `mapstructure:"encryption_key"`
}

// DefaultSecurityConfig 返回默认安全配置
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		ContentTypeOptions:    "nosniff",
		FrameOptions:          "DENY",
		XSSProtection:         "1; mode=block",
		StrictTransport:       "max-age=31536000; includeSubDomains",
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionsPolicy:     "camera=(), microphone=(), geolocation=()",
	}
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Prefix string `mapstructure:"prefix"` // 缓存 key 前缀，用于多实例隔离
}

type ManagementConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	EnablePprof     bool   `mapstructure:"enable_pprof"`
	ProbeTimeoutSec int    `mapstructure:"probe_timeout_sec"`
}

type AuditConfig struct {
	QueueSize       int `mapstructure:"queue_size"`
	BatchSize       int `mapstructure:"batch_size"`
	FlushIntervalMS int `mapstructure:"flush_interval_ms"`
}

type StorageConfig struct {
	Type    string `mapstructure:"type"`     // local, s3, oss, minio
	BaseDir string `mapstructure:"base_dir"` // 本地存储目录
	BaseURL string `mapstructure:"base_url"` // 访问 URL 前缀

	// S3/OSS/MinIO 通用（local 不用）
	Endpoint  string `mapstructure:"endpoint"`   // 如 oss-cn-hangzhou.aliyuncs.com、http://localhost:9000
	Region    string `mapstructure:"region"`     // 如 us-east-1
	Bucket    string `mapstructure:"bucket"`     // 存储桶
	AccessKey string `mapstructure:"access_key"` // 访问密钥
	SecretKey string `mapstructure:"secret_key"` // 密钥
	PathStyle bool   `mapstructure:"path_style"` // 路径风格（MinIO 必须 true）
}

// UploadConfig 文件上传配置（含安全扫描）
type UploadConfig struct {
	ClamAV ClamAVConfig `mapstructure:"clamav"`
}

// ClamAVConfig ClamAV 病毒扫描配置
type ClamAVConfig struct {
	Enabled    bool   `mapstructure:"enabled"`     // 是否启用，false 时上传不扫描
	Address    string `mapstructure:"address"`     // clamd 监听地址，如 127.0.0.1:3310
	TimeoutSec int    `mapstructure:"timeout_sec"` // 扫描超时（秒），0 用默认 10
}

type HTTPConfig struct {
	Host                 string    `mapstructure:"host"`
	Port                 int       `mapstructure:"port"`
	Mode                 string    `mapstructure:"mode"`
	ReadHeaderTimeoutSec int       `mapstructure:"read_header_timeout_sec"`
	ReadTimeoutSec       int       `mapstructure:"read_timeout_sec"`
	WriteTimeoutSec      int       `mapstructure:"write_timeout_sec"`
	IdleTimeoutSec       int       `mapstructure:"idle_timeout_sec"`
	ShutdownTimeoutSec   int       `mapstructure:"shutdown_timeout_sec"`
	MaxBodyBytes         int64     `mapstructure:"max_body_bytes"`
	TrustedProxies       []string  `mapstructure:"trusted_proxies"`
	AllowedOrigins       []string  `mapstructure:"allowed_origins"`
	TLS                  TLSConfig `mapstructure:"tls"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`   // 是否启用 TLS
	CertFile string `mapstructure:"cert_file"` // 证书文件路径
	KeyFile  string `mapstructure:"key_file"`  // 私钥文件路径
}

type DBConfig struct {
	Driver             string `mapstructure:"driver"` // 数据库驱动: mysql, postgres, mariadb (默认 mysql)
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	User               string `mapstructure:"user"`
	Password           string `mapstructure:"password"`
	Database           string `mapstructure:"database"`
	MaxOpen            int    `mapstructure:"max_open"`
	MaxIdle            int    `mapstructure:"max_idle"`
	ConnMaxLifetimeSec int    `mapstructure:"conn_max_lifetime_sec"`
	ConnMaxIdleTimeSec int    `mapstructure:"conn_max_idle_time_sec"`
	MaxRetries         int    `mapstructure:"max_retries"`
	RetryIntervalSec   int    `mapstructure:"retry_interval_sec"`
	// 读写分离
	ReadHosts []string `mapstructure:"read_hosts"` // 从库地址列表
	ReadPorts []int    `mapstructure:"read_ports"` // 从库端口列表
}

type RedisConfig struct {
	Mode string `mapstructure:"mode"` // single / sentinel / cluster，默认 single

	// 单机模式
	Addr string `mapstructure:"addr"`

	// 哨兵模式
	MasterName       string   `mapstructure:"master_name"`       // 哨兵 master 名称
	SentinelAddrs    []string `mapstructure:"sentinel_addrs"`    // 哨兵节点地址列表
	SentinelPassword string   `mapstructure:"sentinel_password"` // 哨兵节点密码（可选）

	// 集群模式
	ClusterAddrs []string `mapstructure:"cluster_addrs"` // 集群节点地址列表

	Password         string `mapstructure:"password"`
	DB               int    `mapstructure:"db"`
	PoolSize         int    `mapstructure:"pool_size"`
	MinIdleConns     int    `mapstructure:"min_idle_conns"`
	ReadTimeoutSec   int    `mapstructure:"read_timeout_sec"`
	WriteTimeoutSec  int    `mapstructure:"write_timeout_sec"`
	MaxRetries       int    `mapstructure:"max_retries"`
	RetryIntervalSec int    `mapstructure:"retry_interval_sec"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"` // 保留文件数
	MaxAge     int    `mapstructure:"max_age"`     // 保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩
}

type AuthConfig struct {
	JWTSecret             string `mapstructure:"jwt_secret"`
	JWTPreviousSecret     string `mapstructure:"jwt_previous_secret"`
	Issuer                string `mapstructure:"issuer"`
	AccessExpireMin       int    `mapstructure:"access_expire_min"`
	RefreshExpireDay      int    `mapstructure:"refresh_expire_day"`
	PublicRegistration    bool   `mapstructure:"public_registration"`
	LoginRateLimit        int    `mapstructure:"login_rate_limit"`
	LoginRateWindowSec    int    `mapstructure:"login_rate_window_sec"`
	RegisterRateLimit     int    `mapstructure:"register_rate_limit"`
	RegisterRateWindowSec int    `mapstructure:"register_rate_window_sec"`
	ResetCodeTTLMin       int    `mapstructure:"reset_code_ttl_min"` // 密码重置验证码有效期（分钟）
}

// Load 加载配置
// 优先级：环境变量 > .env > app.{env}.yaml > app.yaml
func Load() (*Config, error) {
	v, err := buildViper()
	if err != nil {
		return nil, err
	}
	return unmarshalConfig(v)
}

// buildViper 构造并读取配置的 viper 实例（含环境覆盖文件合并）
func buildViper() (*viper.Viper, error) {
	env := os.Getenv("APP_ENV")

	v := viper.New()

	// 注意：.env 文件由 docker-compose 的 env_file 注入为环境变量
	// 不需要 viper 加载 .env，避免与 docker-compose 冲突

	// 加载 yaml 配置
	v.SetConfigName("app")
	v.SetConfigType("yaml")

	// 查找项目根目录下的 configs/
	wd, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		cfgDir := filepath.Join(wd, "configs")
		if _, err := os.Stat(cfgDir); err == nil {
			v.AddConfigPath(cfgDir)
			break
		}
		wd = filepath.Dir(wd)
	}
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read app.yaml: %w", err)
	}

	// 环境配置覆盖（如 app.prod.yaml）
	if env != "" && env != "dev" {
		v.SetConfigName("app." + env)
		if err := v.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to merge app.%s.yaml: %w", env, err)
			}
		}
	}

	return v, nil
}

// unmarshalConfig 从 viper 实例反序列化并校验配置
func unmarshalConfig(v *viper.Viper) (*Config, error) {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 敏感配置环境变量覆盖（不使用 JIMU_ 前缀）
	applyEnvOverrides(&cfg)

	if err := cfg.Validate(os.Getenv("APP_ENV")); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Watch 监听配置文件变更，变更后重新加载并校验配置，再调用 onChange 应用。
// 注意：仅在 onChange 中应用「运行时安全」的设置（如 log.level）；
// DB/Redis 连接池、监听端口等结构类变更需重启进程才会生效。
func Watch(onChange func(*Config) error) error {
	v, err := buildViper()
	if err != nil {
		return err
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		cfg, err := unmarshalConfig(v)
		if err != nil {
			log.Printf("config reload failed: %v", err)
			return
		}
		if err := onChange(cfg); err != nil {
			log.Printf("apply reloaded config failed: %v", err)
		}
	})
	return nil
}

// Dialect 返回用于 goose 迁移的方言名称
func (c DBConfig) Dialect() string {
	switch strings.ToLower(c.Driver) {
	case "postgres", "postgresql":
		return "postgres"
	case "mariadb", "mysql":
		return "mysql"
	default:
		return "mysql"
	}
}

// IsPostgres 是否 PostgreSQL
func (c DBConfig) IsPostgres() bool {
	return c.Dialect() == "postgres"
}

// applyEnvOverrides 应用环境变量覆盖（简洁命名，无 JIMU_ 前缀）
// 支持 _FILE 后缀从文件读取敏感值（Docker Secrets 兼容）
func applyEnvOverrides(cfg *Config) {
	// 认证：优先 JWT_SECRET_FILE，其次 JWT_SECRET
	if v := getEnvOrFile("JWT_SECRET_FILE", "JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	// 密钥轮换：旧 JWT 密钥（用于验证轮换期间尚未过期的旧 token）
	if v := getEnvOrFile("JWT_PREVIOUS_SECRET_FILE", "JWT_PREVIOUS_SECRET"); v != "" {
		cfg.Auth.JWTPreviousSecret = v
	}
	// 数据库
	if v := os.Getenv("DB_DRIVER"); v != "" {
		cfg.DB.Driver = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.DB.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.DB.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.DB.User = v
	}
	// 密码：优先 DB_PASSWORD_FILE，其次 DB_PASSWORD
	if v := getEnvOrFile("DB_PASSWORD_FILE", "DB_PASSWORD"); v != "" {
		cfg.DB.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.DB.Database = v
	}
	// Redis
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := getEnvOrFile("REDIS_PASSWORD_FILE", "REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = db
		}
	}
	// Management 端点监听地址（容器内需暴露给 Prometheus 抓取）
	if v := os.Getenv("MANAGEMENT_HOST"); v != "" {
		cfg.Management.Host = v
	}
	if v := os.Getenv("MANAGEMENT_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Management.Port = port
		}
	}
	// HTTP 监听端口（多实例/测试用）
	if v := os.Getenv("HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.HTTP.Port = port
		}
	}
	// 字段级加密密钥
	if v := getEnvOrFile("ENCRYPTION_KEY_FILE", "ENCRYPTION_KEY"); v != "" {
		cfg.Security.EncryptionKey = v
	}
}

// getEnvOrFile 优先从 _FILE 指向的文件读取，其次直接读取环境变量
func getEnvOrFile(fileKey, directKey string) string {
	// 优先从文件读取（Docker Secrets）
	if path := os.Getenv(fileKey); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			// 去除首尾空白（secret 文件通常有末尾换行符）
			return strings.TrimSpace(string(data))
		}
	}
	// 回退到直接环境变量
	return os.Getenv(directKey)
}

func contains(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
