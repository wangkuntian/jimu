package heavydeps

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOfMatchesModulePrefix 前缀匹配按「模块路径或其后接 /」判定：命中返回展示名，
// 未命中返回空串，避免 github.com/xuri/excelize-foo 之类的同前缀异模块被误判。
func TestOfMatchesModulePrefix(t *testing.T) {
	assert.Equal(t, "kafka-go", Of("github.com/segmentio/kafka-go/protocol"))
	assert.Equal(t, "excelize", Of("github.com/xuri/excelize/v2"))
	assert.Equal(t, "aws-sdk-go-v2", Of("github.com/aws/aws-sdk-go-v2/service/s3"))
	assert.Equal(t, "amqp091-go", Of("github.com/rabbitmq/amqp091-go"))
	assert.Equal(t, "", Of("github.com/gin-gonic/gin"))
	// 同前缀异模块必须落空：只有「模块路径本身或其后接 /」才算命中。
	assert.Equal(t, "", Of("github.com/xuri/excelize-foo"))
	assert.Equal(t, "", Of("github.com/segmentio/kafka-go-extras"))
	assert.Equal(t, []string{"amqp091-go", "aws-sdk-go-v2", "excelize", "kafka-go"}, Names())
}
