package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	svc := NewService("v1.0.0", "test", nil)
	assert.Equal(t, "v1.0.0", svc.Version())
	assert.Equal(t, "test", svc.Env())
	assert.False(t, svc.startTime.IsZero())
}
