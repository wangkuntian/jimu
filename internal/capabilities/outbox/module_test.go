package outbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModuleDescriptor 描述符：软依赖 queue（publisher=mq 才需要），自有 outbox_events 表。
func TestModuleDescriptor(t *testing.T) {
	d := Descriptor
	assert.Equal(t, []string{"queue"}, d.SoftRequires)
	assert.ElementsMatch(t, []string{"outbox_events"}, d.Owns)
}
