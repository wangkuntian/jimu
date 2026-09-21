package queue

import (
	"strings"
	"testing"

	"jimu/internal/config"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type viperSection struct{ v *viper.Viper }

func (s viperSection) UnmarshalKey(key string, rawVal any) error {
	return s.v.UnmarshalKey(key, rawVal)
}

func TestConfigKeysAreStable(t *testing.T) {
	assert.Equal(t, "queue", ConfigKey, "对外配置键不得变化")
	assert.Equal(t, "scheduler", SchedulerConfigKey, "对外配置键不得变化")
}

// TestDescriptorDeclaresConfigSections 描述符必须声明 queue 与 scheduler 两段，
// 否则框架不会加载/校验它们。
func TestDescriptorDeclaresConfigSections(t *testing.T) {
	declared := map[string]bool{}
	for _, spec := range Descriptor.Configs {
		require.NotNil(t, spec.New)
		_, ok := spec.New().(config.SectionConfig)
		require.True(t, ok, "段实例必须实现 config.SectionConfig")
		declared[spec.Section] = true
	}
	require.True(t, declared[ConfigKey], "Descriptor 必须声明配置段 %q", ConfigKey)
	require.True(t, declared[SchedulerConfigKey], "Descriptor 必须声明配置段 %q", SchedulerConfigKey)
}

// loadQueue / loadScheduler 走框架同款机制（config.LoadSection：解码 → 默认值 → 校验）。
func loadQueue(dec config.SectionDecoder) (*Config, error) {
	var c Config
	if err := config.LoadSection(dec, ConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func loadScheduler(dec config.SectionDecoder) (*SchedulerConfig, error) {
	var c SchedulerConfig
	if err := config.LoadSection(dec, SchedulerConfigKey, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// TestValidateQueueType 迁移自 internal/config 的 queue.type 校验。
func TestValidateQueueType(t *testing.T) {
	for _, tp := range []Type{TypeRedis, TypeKafka, TypeRabbitMQ} {
		require.NoError(t, (&Config{Type: tp}).Validate())
	}
	err := (&Config{Type: "nats"}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue.type")
}

// TestValidateSchedulerStore 迁移自 internal/config 的同名用例。
func TestValidateSchedulerStore(t *testing.T) {
	require.NoError(t, (&SchedulerConfig{Store: SchedulerStoreMemory}).Validate())
	require.NoError(t, (&SchedulerConfig{Store: SchedulerStoreMySQL}).Validate())

	err := (&SchedulerConfig{Store: "etcd"}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheduler.store")
}

// TestSupportsOutboxMQ 跨能力约束的判定表（outbox.publisher=mq 的队列类型白名单）。
func TestSupportsOutboxMQ(t *testing.T) {
	assert.True(t, SupportsOutboxMQ(TypeRedis))
	assert.True(t, SupportsOutboxMQ(TypeKafka))
	assert.True(t, SupportsOutboxMQ(TypeRabbitMQ))
	assert.False(t, SupportsOutboxMQ(Type("nats")))
	assert.False(t, SupportsOutboxMQ(Type("")))
}

// TestLoadMapsYAMLKeys YAML 键名与下沉前逐一一致（含嵌套 kafka/rabbitmq 段）。
func TestLoadMapsYAMLKeys(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(stringsReader(`
queue:
  type: "kafka"
  kafka:
    brokers: ["kafka:9092"]
    topic: "jimu"
    group_id: "g1"
    max_retry: 3
  rabbitmq:
    url: "amqp://guest:guest@rabbitmq:5672/"
    queue: "jimu-q"
    exchange: "jimu-x"
`)))
	cfg, err := loadQueue(viperSection{v: v})
	require.NoError(t, err)
	assert.Equal(t, TypeKafka, cfg.Type)
	assert.Equal(t, []string{"kafka:9092"}, cfg.Kafka.Brokers)
	assert.Equal(t, "jimu", cfg.Kafka.Topic)
	assert.Equal(t, "g1", cfg.Kafka.GroupID)
	assert.Equal(t, 3, cfg.Kafka.MaxRetry)
	assert.Equal(t, "amqp://guest:guest@rabbitmq:5672/", cfg.RabbitMQ.URL)
	assert.Equal(t, "jimu-q", cfg.RabbitMQ.QueueName, "YAML 键沿用 queue")
	assert.Equal(t, "jimu-x", cfg.RabbitMQ.Exchange)
}

// TestLoadSchedulerSection 调度器段独立解码。
func TestLoadSchedulerSection(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(stringsReader("scheduler:\n  store: \"mysql\"\n")))
	cfg, err := loadScheduler(viperSection{v: v})
	require.NoError(t, err)
	assert.Equal(t, SchedulerStoreMySQL, cfg.Store)
}

func stringsReader(s string) *strings.Reader { return strings.NewReader(s) }
