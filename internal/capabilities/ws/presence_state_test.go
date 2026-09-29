package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresenceIsOnline(t *testing.T) {
	assert.True(t, (&Presence{Status: StatusOnline}).IsOnline())
	assert.False(t, (&Presence{Status: StatusOffline}).IsOnline())
}

func TestPresenceIsTyping(t *testing.T) {
	// 离线永不 typing
	assert.False(t, (&Presence{Status: StatusOffline, LastTyping: time.Now()}).IsTyping())

	// 最近有输入 → true
	p := &Presence{Status: StatusOnline, LastTyping: time.Now().Add(-1 * time.Second)}
	assert.True(t, p.IsTyping())

	// 超过超时 → false
	p = &Presence{Status: StatusOnline, LastTyping: time.Now().Add(-TypingTimeout - time.Second)}
	assert.False(t, p.IsTyping())

	// 从未输入 → false
	p = &Presence{Status: StatusOnline}
	assert.False(t, p.IsTyping())
}

func TestPresenceStale(t *testing.T) {
	fresh := &Presence{Status: StatusOnline, LastHeartbeat: time.Now()}
	assert.False(t, fresh.Stale())

	stale := &Presence{Status: StatusOnline, LastHeartbeat: time.Now().Add(-HeartbeatInterval*time.Duration(MaxMissedHeartbeats) - time.Second)}
	assert.True(t, stale.Stale())
}

func TestPresenceManagerHeartbeat(t *testing.T) {
	pm := NewPresenceManager()
	pm.Online(1, "c1")

	p, ok := pm.GetPresence(1)
	require.True(t, ok)
	before := p.LastHeartbeat
	time.Sleep(2 * time.Millisecond)

	pm.Heartbeat(1)
	p, ok = pm.GetPresence(1)
	require.True(t, ok)
	assert.True(t, p.LastHeartbeat.After(before))

	// 未注册用户心跳不 panic
	pm.Heartbeat(999)
}

func TestPresenceManagerHeartbeatRestoresOnlineStatus(t *testing.T) {
	pm := NewPresenceManager()
	// 白盒构造一个非 online 状态的 presence
	pm.presences[5] = &Presence{UserID: 5, Status: "busy", LastHeartbeat: time.Now()}

	pm.Heartbeat(5)
	p, ok := pm.GetPresence(5)
	require.True(t, ok)
	assert.Equal(t, StatusOnline, p.Status)
	assert.Equal(t, 0, p.MissedBeats)
}

func TestPresenceManagerSetTyping(t *testing.T) {
	pm := NewPresenceManager()
	// 未注册用户不 panic
	pm.SetTyping(999)

	pm.Online(1, "c1")
	pm.SetTyping(1)
	assert.True(t, pm.IsOnline(1))
	p, ok := pm.GetPresence(1)
	require.True(t, ok)
	assert.True(t, p.IsTyping())
}

func TestPresenceManagerOfflineMissing(t *testing.T) {
	pm := NewPresenceManager()
	pm.Offline(999) // 不 panic
	assert.Equal(t, 0, pm.OnlineCount())
}

func TestPresenceManagerOnlineCountFiltersOffline(t *testing.T) {
	pm := NewPresenceManager()
	pm.Online(1, "c1")
	pm.Online(2, "c2")
	// 白盒：状态非 online 的不计数
	pm.presences[3] = &Presence{UserID: 3, Status: StatusOffline, LastHeartbeat: time.Now()}

	assert.Equal(t, 2, pm.OnlineCount())
	assert.ElementsMatch(t, []uint64{1, 2}, pm.OnlineUsers())
}

func TestPresenceManagerStaleUsers(t *testing.T) {
	pm := NewPresenceManager()
	pm.Online(1, "c1")
	pm.Online(2, "c2")

	// 白盒：把 1 号心跳改旧，2 号保持新鲜
	p1, ok := pm.GetPresence(1)
	require.True(t, ok)
	p1.LastHeartbeat = time.Now().Add(-HeartbeatInterval*time.Duration(MaxMissedHeartbeats) - time.Second)

	assert.ElementsMatch(t, []uint64{1}, pm.StaleUsers())

	// 离线用户不计入 stale
	pm.Offline(1)
	assert.Empty(t, pm.StaleUsers())
}

func TestPresenceManagerAllPresences(t *testing.T) {
	pm := NewPresenceManager()
	pm.Online(1, "c1")
	pm.Online(2, "c2")

	pres := pm.AllPresences()
	assert.Len(t, pres, 2)
	seen := map[uint64]bool{}
	for _, p := range pres {
		seen[p.UserID] = true
	}
	assert.True(t, seen[1] && seen[2])
}
