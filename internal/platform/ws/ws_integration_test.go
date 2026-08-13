package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"jimu/internal/platform/auth"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wsFixture 管理一个完整的 ws 测试环境
type wsFixture struct {
	server   *httptest.Server
	hub      *ClientHub
	jwt      *auth.JWT
	presence *PresenceManager
	channels *ChannelManager
}

func newWSFixture(t *testing.T) *wsFixture {
	t.Helper()
	jwt := auth.New("test-secret", "test-issuer", 60, 7)
	pm := NewPresenceManager()
	cm := NewChannelManager()
	hub := NewClientHub(pm, cm)

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	srv := httptest.NewServer(WSHandler(hub, jwt, pm, cm))
	t.Cleanup(func() {
		cancel()
		srv.Close()
	})
	return &wsFixture{server: srv, hub: hub, jwt: jwt, presence: pm, channels: cm}
}

func (f *wsFixture) dial(t *testing.T, userID uint64) (*websocket.Conn, *testConn) {
	t.Helper()
	token, err := f.jwt.GenerateAccess(userID, "sess-"+strconv.FormatUint(userID, 10))
	require.NoError(t, err)
	u := "ws" + strings.TrimPrefix(f.server.URL, "http") + "?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn, newTestConn(t, conn)
}

// errConnClosed 表示连接已关闭，不能再读取
var errConnClosed = errors.New("ws conn closed")

// testConn 包装客户端连接，按 '\n' 拆分服务器批量帧
type testConn struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	q      [][]byte
	closed bool
}

func newTestConn(t *testing.T, conn *websocket.Conn) *testConn {
	t.Helper()
	tc := &testConn{conn: conn}
	t.Cleanup(func() { tc.drain() })
	return tc
}

// drain 读取并丢弃剩余帧（清理用），失败后立即停止，避免对已失败连接重复读取
func (tc *testConn) drain() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.closed {
		return
	}
	_ = tc.conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	for {
		if _, _, err := tc.conn.ReadMessage(); err != nil {
			tc.closed = true
			return
		}
	}
}

// readFrame 返回下一条 JSON 数据。
// 返回 (nil, nil) 表示暂时无数据（读超时）；返回 (nil, err) 表示连接失败，之后不再读取。
func (tc *testConn) readFrame() ([]byte, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.closed {
		return nil, errConnClosed
	}
	if len(tc.q) > 0 {
		d := tc.q[0]
		tc.q = tc.q[1:]
		return d, nil
	}
	_ = tc.conn.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := tc.conn.ReadMessage()
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, nil
		}
		tc.closed = true
		return nil, err
	}
	var first []byte
	for _, p := range bytes.Split(data, []byte{'\n'}) {
		p = bytes.TrimSpace(p)
		if len(p) == 0 {
			continue
		}
		if first == nil {
			first = p
		} else {
			tc.q = append(tc.q, p)
		}
	}
	return first, nil
}

// waitFor 轮询等待指定类型消息
func (tc *testConn) waitFor(t *testing.T, msgType string) *WSMessage {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, err := tc.readFrame()
		if err != nil || data == nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		var m WSMessage
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if m.Type == msgType {
			return &m
		}
	}
	t.Fatalf("timed out waiting for message type %q", msgType)
	return nil
}

// waitForTyping 轮询等待 typing 状态 presence 消息（在线时会收到多条 presence）
func (tc *testConn) waitForTyping(t *testing.T) PresencePayload {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, err := tc.readFrame()
		if err != nil || data == nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		var m WSMessage
		if json.Unmarshal(data, &m) != nil || m.Type != TypePresence {
			continue
		}
		var p PresencePayload
		if json.Unmarshal(m.Payload, &p) == nil && p.Status == StatusTyping {
			return p
		}
	}
	t.Fatalf("timed out waiting for typing presence")
	return PresencePayload{}
}

// assertNoMessage 断言在窗口期内不收到指定类型消息
func (tc *testConn) assertNoMessage(t *testing.T, msgType string, wait time.Duration) {
	t.Helper()
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		data, err := tc.readFrame()
		if err != nil || data == nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		var m WSMessage
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if m.Type == msgType {
			t.Fatalf("unexpected message type %q received", msgType)
		}
	}
}

func writeJSON(t *testing.T, conn *websocket.Conn, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, data))
}

// lastHeartbeat 在锁内读取心跳时间，避免与 Heartbeat 写入竞争
func (f *wsFixture) lastHeartbeat(userID uint64) (time.Time, bool) {
	f.presence.mu.RLock()
	defer f.presence.mu.RUnlock()
	p, ok := f.presence.presences[userID]
	if !ok {
		return time.Time{}, false
	}
	return p.LastHeartbeat, true
}

func waitCond(t *testing.T, msg string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out: " + msg)
}

func TestWSHandlerMissingToken(t *testing.T) {
	f := newWSFixture(t)
	resp, err := http.Get(f.server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWSHandlerInvalidToken(t *testing.T) {
	f := newWSFixture(t)
	resp, err := http.Get(f.server.URL + "?token=bad-token")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWSConnectRegisters(t *testing.T) {
	f := newWSFixture(t)
	_, tc := f.dial(t, 42)

	// 注册后自动订阅用户频道和广播频道
	tc.waitFor(t, TypePresence)
	waitCond(t, "online count", func() bool { return f.hub.OnlineCount() == 1 })
	waitCond(t, "presence online", func() bool { return f.presence.IsOnline(42) })

	assert.Equal(t, 1, f.hub.GetUserConnections(42))
	assert.Equal(t, []uint64{42}, f.hub.OnlineUsers())

	// 注册后自动订阅用户个人频道和广播频道
	cid := connID(f.hub, 42)
	assert.NotEmpty(t, cid)
	_, hasBroadcast := f.hub.channels.GetChannel(ChannelBroadcast)
	assert.True(t, hasBroadcast)
	ch, ok := f.hub.channels.GetChannel(BuildUserChannel(42))
	require.True(t, ok)
	assert.True(t, ch.IsSubscribed(cid))
}

func TestWSPingPong(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence) // 确保连接完成

	writeJSON(t, conn, map[string]interface{}{
		"type":    TypePing,
		"channel": "broadcast",
	})
	pong := tc.waitFor(t, TypePong)
	assert.Equal(t, "broadcast", pong.Channel)
}

func TestWSPongHeartbeat(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	before, ok := f.lastHeartbeat(42)
	require.True(t, ok)
	time.Sleep(2 * time.Millisecond)

	// 客户端发送 pong 控制帧触发服务器 pong handler
	require.NoError(t, conn.WriteControl(websocket.PongMessage, nil, time.Now().Add(time.Second)))

	waitCond(t, "heartbeat updated", func() bool {
		cur, ok := f.lastHeartbeat(42)
		return ok && cur.After(before)
	})

	// 连接仍可用
	writeJSON(t, conn, map[string]interface{}{"type": TypePing})
	tc.waitFor(t, TypePong)
}

func TestWSSubscribeBroadcast(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	writeJSON(t, conn, map[string]interface{}{
		"type":    TypeSubscribe,
		"payload": map[string]interface{}{"channels": []string{"room:abc"}},
	})
	waitCond(t, "subscribed", func() bool {
		ch, ok := f.hub.channels.GetChannel("room:abc")
		return ok && ch.IsSubscribed(connID(f.hub, 42))
	})

	msg, _ := NewMessage(TypeNotification, "room:abc", map[string]string{"title": "hi"})
	f.hub.BroadcastToChannel("room:abc", msg)

	got := tc.waitFor(t, TypeNotification)
	assert.Equal(t, "room:abc", got.Channel)
}

func TestWSSubscribeDenied(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	// 订阅他人用户频道被拒
	writeJSON(t, conn, map[string]interface{}{
		"type":    TypeSubscribe,
		"payload": map[string]interface{}{"channels": []string{"user:999"}},
	})
	errMsg := tc.waitFor(t, "error")
	var payload map[string]string
	require.NoError(t, json.Unmarshal(errMsg.Payload, &payload))
	assert.Contains(t, payload["error"], "user:999")

	// 被拒频道广播不应送达
	msg, _ := NewMessage(TypeNotification, "user:999", map[string]string{"x": "y"})
	f.hub.BroadcastToChannel("user:999", msg)
	tc.assertNoMessage(t, TypeNotification, 400*time.Millisecond)
}

func TestWSSubscribeInvalidPayload(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	writeJSON(t, conn, map[string]interface{}{"type": TypeSubscribe})
	tc.waitFor(t, "error")
}

func TestWSUnsubscribe(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	writeJSON(t, conn, map[string]interface{}{
		"type":    TypeSubscribe,
		"payload": map[string]interface{}{"channels": []string{"room:x"}},
	})
	writeJSON(t, conn, map[string]interface{}{
		"type":    TypeUnsubscribe,
		"payload": map[string]interface{}{"channels": []string{"room:x"}},
	})
	waitCond(t, "unsubscribed", func() bool {
		ch, ok := f.hub.channels.GetChannel("room:x")
		return ok && !ch.IsSubscribed(connID(f.hub, 42))
	})

	msg, _ := NewMessage(TypeNotification, "room:x", map[string]string{"x": "y"})
	f.hub.BroadcastToChannel("room:x", msg)
	tc.assertNoMessage(t, TypeNotification, 400*time.Millisecond)
}

func TestWSUnsubscribeInvalidPayload(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	writeJSON(t, conn, map[string]interface{}{"type": TypeUnsubscribe})
	tc.waitFor(t, "error")
}

func TestWSTyping(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	writeJSON(t, conn, map[string]interface{}{
		"type": TypePresence,
		"payload": map[string]interface{}{
			"user_id": 42,
			"status":  StatusTyping,
		},
	})

	// 服务端把 presence 广播到用户频道和广播频道，客户端可能先收到重复的 online，
	// 需轮询直到收到 typing
	p := tc.waitForTyping(t)
	assert.Equal(t, StatusTyping, p.Status)

	waitCond(t, "typing state", func() bool {
		f.presence.mu.RLock()
		defer f.presence.mu.RUnlock()
		p, ok := f.presence.presences[42]
		return ok && p.IsTyping()
	})
}

func TestWSChatBroadcast(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	writeJSON(t, conn, map[string]interface{}{
		"type":    TypeSubscribe,
		"payload": map[string]interface{}{"channels": []string{"room:chat"}},
	})
	waitCond(t, "subscribed", func() bool {
		ch, ok := f.hub.channels.GetChannel("room:chat")
		return ok && ch.IsSubscribed(connID(f.hub, 42))
	})

	writeJSON(t, conn, map[string]interface{}{
		"type":    TypeChat,
		"channel": "room:chat",
		"payload": map[string]interface{}{"from": 42, "to": "room:chat", "content": "hello"},
	})

	got := tc.waitFor(t, TypeChat)
	assert.Equal(t, "room:chat", got.Channel)
	var chat ChatPayload
	require.NoError(t, json.Unmarshal(got.Payload, &chat))
	assert.Equal(t, "hello", chat.Content)
}

func TestWSChatInvalidPayload(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	writeJSON(t, conn, map[string]interface{}{"type": TypeChat, "channel": "room:chat"})
	tc.waitFor(t, "error")
}

func TestWSInvalidJSON(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("not-json")))
	tc.waitFor(t, "error")
}

func TestWSUnknownType(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	writeJSON(t, conn, map[string]interface{}{"type": "foo"})
	tc.waitFor(t, "error")
}

func TestWSSendToUser(t *testing.T) {
	f := newWSFixture(t)
	_, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)

	msg, _ := NewMessage(TypeNotification, BuildUserChannel(42), map[string]string{"title": "direct"})
	f.hub.SendToUser(42, msg)

	got := tc.waitFor(t, TypeNotification)
	assert.Equal(t, "direct", mustDecodeTitle(t, got))

	// 发给其他用户不送达
	other, _ := NewMessage(TypeNotification, BuildUserChannel(1), map[string]string{"title": "other"})
	f.hub.SendToUser(1, other)
	tc.assertNoMessage(t, TypeNotification, 300*time.Millisecond)
}

func TestWSBroadcastAll(t *testing.T) {
	f := newWSFixture(t)
	_, tc1 := f.dial(t, 1)
	_, tc2 := f.dial(t, 2)
	tc1.waitFor(t, TypePresence)
	tc2.waitFor(t, TypePresence)

	waitCond(t, "two conns", func() bool { return f.hub.OnlineCount() == 2 })

	msg, _ := NewMessage(TypeNotification, ChannelBroadcast, map[string]string{"title": "all"})
	f.hub.Broadcast(msg)

	tc1.waitFor(t, TypeNotification)
	tc2.waitFor(t, TypeNotification)
}

func TestWSSendToUsers(t *testing.T) {
	f := newWSFixture(t)
	_, tc1 := f.dial(t, 1)
	_, tc2 := f.dial(t, 2)
	tc1.waitFor(t, TypePresence)
	tc2.waitFor(t, TypePresence)

	msg, _ := NewMessage(TypeNotification, "broadcast", map[string]string{"title": "batch"})
	f.hub.SendToUsers([]uint64{1, 2}, msg)

	tc1.waitFor(t, TypeNotification)
	tc2.waitFor(t, TypeNotification)
}

func TestWSDisconnectUnregisters(t *testing.T) {
	f := newWSFixture(t)
	conn, tc := f.dial(t, 42)
	tc.waitFor(t, TypePresence)
	waitCond(t, "registered", func() bool { return f.hub.OnlineCount() == 1 })

	require.NoError(t, conn.Close())
	waitCond(t, "unregistered", func() bool {
		return f.hub.OnlineCount() == 0 && !f.presence.IsOnline(42)
	})
	assert.Equal(t, 0, f.hub.GetUserConnections(42))
}

func TestWSMultipleConnections(t *testing.T) {
	f := newWSFixture(t)
	conn1, tc1 := f.dial(t, 42)
	_, tc2 := f.dial(t, 42)
	tc1.waitFor(t, TypePresence)
	tc2.waitFor(t, TypePresence)

	waitCond(t, "two conns same user", func() bool { return f.hub.GetUserConnections(42) == 2 })
	// 多端登录下用户仍在线
	assert.True(t, f.presence.IsOnline(42))

	// 关闭一端仍在线
	require.NoError(t, conn1.Close())
	waitCond(t, "one conn left", func() bool { return f.hub.GetUserConnections(42) == 1 })
	assert.True(t, f.presence.IsOnline(42))

	// 关闭另一端下线
	tc2.conn.Close()
	waitCond(t, "all gone", func() bool { return f.hub.OnlineCount() == 0 && !f.presence.IsOnline(42) })
}

// connID 从 hub 中查询某用户的连接 ID（仅测试用）
func connID(hub *ClientHub, userID uint64) string {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for id := range hub.userIndex[userID] {
		return id
	}
	return ""
}

func mustDecodeTitle(t *testing.T, msg *WSMessage) string {
	t.Helper()
	var payload map[string]string
	require.NoError(t, json.Unmarshal(msg.Payload, &payload))
	return payload["title"]
}

func TestClientChannels(t *testing.T) {
	c := &Client{channels: map[string]bool{"user:1": true, "room:x": true}}
	assert.ElementsMatch(t, []string{"user:1", "room:x"}, c.Channels())
}

func TestHubSafeUnregisterWhenBusy(t *testing.T) {
	pm := NewPresenceManager()
	cm := NewChannelManager()
	hub := NewClientHub(pm, cm)

	// 填满 unregister 通道，强制走 default 分支
	for i := 0; i < cap(hub.unregister); i++ {
		hub.unregister <- &Client{}
	}

	serverConn, clientConn := newUpgradedConnPair(t)
	_ = clientConn

	client := &Client{
		hub:      hub,
		conn:     serverConn,
		send:     make(chan []byte, 4),
		userID:   7,
		connID:   "c7",
		channels: map[string]bool{"user:7": true},
	}
	hub.mu.Lock()
	hub.clients["c7"] = client
	hub.userIndex[7] = map[string]bool{"c7": true}
	hub.mu.Unlock()

	hub.safeUnregister(client)

	assert.Equal(t, 0, hub.OnlineCount())
	assert.Equal(t, 0, hub.GetUserConnections(7))
}

func TestHubSafeUnregisterNotRegistered(t *testing.T) {
	pm := NewPresenceManager()
	cm := NewChannelManager()
	hub := NewClientHub(pm, cm)
	for i := 0; i < cap(hub.unregister); i++ {
		hub.unregister <- &Client{}
	}

	serverConn, _ := newUpgradedConnPair(t)
	client := &Client{
		hub:    hub,
		conn:   serverConn,
		send:   make(chan []byte, 4),
		userID: 8,
		connID: "c8",
	}

	// 未在 map 中，直接关闭连接不 panic
	hub.safeUnregister(client)
	assert.Equal(t, 0, hub.OnlineCount())
}

// newUpgradedConnPair 建立一对真实 websocket 连接（服务端 + 客户端）
func newUpgradedConnPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	done := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		done <- c
	}))
	t.Cleanup(srv.Close)

	u := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	server := <-done
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = server.Close() })
	return server, client
}
