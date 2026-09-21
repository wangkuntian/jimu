package queue

import (
	"strings"
	"testing"

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
	cfg, err := Load(viperSection{v: v})
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
	cfg, err := LoadScheduler(viperSection{v: v})
	require.NoError(t, err)
	assert.Equal(t, SchedulerStoreMySQL, cfg.Store)
}

func stringsReader(s string) *strings.Reader { return strings.NewReader(s) }
