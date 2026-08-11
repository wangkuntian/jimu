package ws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPresenceOnlineOffline(t *testing.T) {
	pm := NewPresenceManager()
	pm.Online(1, "c1")
	assert.True(t, pm.IsOnline(1))
	assert.Equal(t, 1, pm.OnlineCount())

	pm.Offline(1)
	assert.False(t, pm.IsOnline(1))
	assert.Equal(t, 0, pm.OnlineCount())
}

func TestPresenceGet(t *testing.T) {
	pm := NewPresenceManager()
	pm.Online(42, "c42")
	p, ok := pm.GetPresence(42)
	assert.True(t, ok)
	assert.Equal(t, StatusOnline, p.Status)
	assert.Equal(t, uint64(42), p.UserID)
}

func TestPresenceOnlineUsers(t *testing.T) {
	pm := NewPresenceManager()
	pm.Online(1, "c1")
	pm.Online(2, "c2")
	pm.Offline(2)
	users := pm.OnlineUsers()
	assert.Equal(t, []uint64{1}, users)
}
