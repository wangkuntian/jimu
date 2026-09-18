package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConn struct {
	mu     sync.Mutex
	userID string
	sent   []byte
	closed bool
}

func (m *mockConn) Send(data []byte) error {
	m.mu.Lock()
	m.sent = data
	m.mu.Unlock()
	return nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}

func (m *mockConn) UserID() string { return m.userID }

func (m *mockConn) getSent() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sent
}

func (m *mockConn) getClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestHubLifecycle(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)

	conn := &mockConn{userID: "u1"}
	h.Register("u1", conn)

	waitHub(t, func() bool { return h.OnlineCount() == 1 })
	assert.True(t, h.IsOnline("u1"))

	// 广播给在线用户
	h.SendToUser("u1", Message{Body: "hi"})
	waitHub(t, func() bool { return conn.getSent() != nil })
	assert.Contains(t, string(conn.getSent()), "hi")

	// 批量发送
	h.SendToUsers([]string{"u1", "u2"}, Message{Body: "all"})

	// 注销
	h.Unregister("u1")
	waitHub(t, func() bool { return h.OnlineCount() == 0 && conn.getClosed() })
	assert.False(t, h.IsOnline("u1"))

	cancel()
}

func TestWebSocketNotification(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	conn := &mockConn{userID: "u1"}
	h.Register("u1", conn)
	waitHub(t, func() bool { return h.IsOnline("u1") })

	w := NewWebSocket(h)
	assert.Equal(t, ChannelWebSocket, w.Channel())

	require.NoError(t, w.Send(ctx, Message{To: "u1", Body: "ws"}))
	waitHub(t, func() bool { return conn.getSent() != nil })
	assert.Contains(t, string(conn.getSent()), "ws")

	require.NoError(t, w.SendBatch(ctx, []Message{{To: "u1", Body: "ws2"}}))
}

// waitHub 轮询等待异步 hub 状态
func waitHub(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("hub condition not met")
}
