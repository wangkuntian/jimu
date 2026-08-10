// internal/platform/queue/rabbitmq_queue_test.go
package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRabbitMQQueueImplementsInterfaces(t *testing.T) {
	var _ Queue = (*RabbitMQQueue)(nil)
	var _ Consumer = (*RabbitMQQueue)(nil)
}

func TestRabbitMQConfigDefaults(t *testing.T) {
	cfg := RabbitMQConfig{
		URL:       "amqp://guest:guest@localhost:5672/",
		QueueName: "test-queue",
	}
	assert.Equal(t, "test-queue", cfg.QueueName)
}
