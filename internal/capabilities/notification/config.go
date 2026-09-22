package notification

import "jimu/internal/config"

// 本包在 app.yaml 中拥有的配置段键（三段合归通知包）。
const (
	EmailKey        = "email"
	SMSKey          = "sms"
	NotificationKey = "notification"
)

// EmailSection 邮件通知配置段（字段与 YAML 键名同下沉前一致）。
type EmailSection struct {
	Enabled  bool   `mapstructure:"enabled"`  // 是否启用真实 SMTP 发送；false 时回退日志渠道
	Host     string `mapstructure:"host"`     // SMTP 服务器地址
	Port     int    `mapstructure:"port"`     // SMTP 端口（通常 25/465/587）
	Username string `mapstructure:"username"` // 认证用户名
	Password string `mapstructure:"password"` // 认证密码（敏感，建议环境变量注入）
	From     string `mapstructure:"from"`     // 发件人地址
}

// SMSSection 短信通知配置段（字段与 YAML 键名同下沉前一致）。
type SMSSection struct {
	Enabled   bool   `mapstructure:"enabled"`    // 是否启用真实短信发送；false 时回退日志渠道
	Provider  string `mapstructure:"provider"`   // 短信服务商：aliyun
	APIKey    string `mapstructure:"api_key"`    // AccessKey ID（敏感）
	APISecret string `mapstructure:"api_secret"` // AccessKey Secret（敏感）
	SignName  string `mapstructure:"sign_name"`  // 短信签名
}

// Section 通知总配置段（webhook 渠道）。
type Section struct {
	Webhook WebhookSection `mapstructure:"webhook"`
}

// WebhookSection Webhook 通知配置段。
type WebhookSection struct {
	SignSecret string `mapstructure:"sign_secret"` // 载荷签名密钥（HMAC-SHA256）；空则不签名
}

// Config 本包拥有的三段配置。
type Config struct {
	Email        EmailSection `mapstructure:"email"`
	SMS          SMSSection   `mapstructure:"sms"`
	Notification Section      `mapstructure:"notification"`
}

// 以下 ApplyDefaults/Validate 均为空实现：本包各段无配置层默认值与校验
// （真实渠道未启用时回退日志渠道，由装配侧判定），键与语义同下沉前一致。

func (s *EmailSection) ApplyDefaults()  {}
func (s *EmailSection) Validate() error { return nil }

func (s *SMSSection) ApplyDefaults()  {}
func (s *SMSSection) Validate() error { return nil }

func (s *Section) ApplyDefaults()  {}
func (s *Section) Validate() error { return nil }

// Load 解码本包拥有的三个配置段。通知不属 catalog 能力，由组合根无条件加载。
func Load(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, EmailKey, &c.Email); err != nil {
		return nil, err
	}
	if err := config.LoadSection(dec, SMSKey, &c.SMS); err != nil {
		return nil, err
	}
	if err := config.LoadSection(dec, NotificationKey, &c.Notification); err != nil {
		return nil, err
	}
	return &c, nil
}
