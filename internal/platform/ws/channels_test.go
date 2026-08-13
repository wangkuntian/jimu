package ws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChannel(t *testing.T) {
	ch := NewChannel("user:1", "user")
	assert.Equal(t, "user:1", ch.Name)
	assert.Equal(t, "user", ch.Type)
	assert.Empty(t, ch.Subscribers)
}

func TestChannelSubscribeUnsubscribe(t *testing.T) {
	ch := NewChannel("room:a", "room")
	assert.False(t, ch.IsSubscribed("c1"))
	assert.Equal(t, 0, ch.SubscriberCount())

	ch.Subscribe("c1")
	ch.Subscribe("c2")
	assert.True(t, ch.IsSubscribed("c1"))
	assert.True(t, ch.IsSubscribed("c2"))
	assert.Equal(t, 2, ch.SubscriberCount())

	// 重复订阅幂等
	ch.Subscribe("c1")
	assert.Equal(t, 2, ch.SubscriberCount())

	ch.Unsubscribe("c1")
	assert.False(t, ch.IsSubscribed("c1"))
	assert.Equal(t, 1, ch.SubscriberCount())

	// 取消未订阅的连接无副作用
	ch.Unsubscribe("missing")
	assert.Equal(t, 1, ch.SubscriberCount())
}

func TestChannelManagerPreCreatedBroadcast(t *testing.T) {
	cm := NewChannelManager()
	assert.Equal(t, 1, cm.ChannelCount())

	ch, ok := cm.GetChannel(ChannelBroadcast)
	require.True(t, ok)
	assert.Equal(t, "broadcast", ch.Type)
}

func TestChannelManagerSubscribeCreatesWithType(t *testing.T) {
	cm := NewChannelManager()

	cm.Subscribe("c1", "room:abc")
	ch, ok := cm.GetChannel("room:abc")
	require.True(t, ok)
	assert.Equal(t, "room", ch.Type)
	assert.True(t, ch.IsSubscribed("c1"))

	cm.Subscribe("c2", "user:5")
	ch, ok = cm.GetChannel("user:5")
	require.True(t, ok)
	assert.Equal(t, "user", ch.Type)

	cm.Subscribe("c3", "broadcast")
	ch, ok = cm.GetChannel("broadcast")
	require.True(t, ok)
	assert.Equal(t, "broadcast", ch.Type)
	assert.True(t, ch.IsSubscribed("c3"))

	// broadcast 频道由 NewChannelManager 预创建，订阅不新增
	assert.Equal(t, 3, cm.ChannelCount())
}

func TestChannelManagerSubscribeExisting(t *testing.T) {
	cm := NewChannelManager()
	cm.Subscribe("c1", "room:x")
	cm.Subscribe("c2", "room:x")

	ch, ok := cm.GetChannel("room:x")
	require.True(t, ok)
	assert.Equal(t, 2, ch.SubscriberCount())
	assert.Equal(t, 2, cm.ChannelCount()) // 不重复创建
}

func TestChannelManagerGetSubscribers(t *testing.T) {
	cm := NewChannelManager()
	assert.Nil(t, cm.GetSubscribers("room:missing"))

	cm.Subscribe("c1", "room:y")
	cm.Subscribe("c2", "room:y")
	subs := cm.GetSubscribers("room:y")
	assert.ElementsMatch(t, []string{"c1", "c2"}, subs)
}

func TestChannelManagerUnsubscribe(t *testing.T) {
	cm := NewChannelManager()
	cm.Subscribe("c1", "room:z")
	cm.Unsubscribe("c1", "room:z")
	ch, ok := cm.GetChannel("room:z")
	require.True(t, ok)
	assert.False(t, ch.IsSubscribed("c1"))

	// 未存在频道不 panic
	cm.Unsubscribe("c1", "room:nope")
}

func TestChannelManagerUnsubscribeAll(t *testing.T) {
	cm := NewChannelManager()
	cm.Subscribe("c1", "room:a")
	cm.Subscribe("c1", "room:b")
	cm.Subscribe("c2", "room:a")

	cm.UnsubscribeAll("c1", []string{"room:a", "room:b"})

	chA, _ := cm.GetChannel("room:a")
	chB, _ := cm.GetChannel("room:b")
	assert.False(t, chA.IsSubscribed("c1"))
	assert.True(t, chA.IsSubscribed("c2"))
	assert.False(t, chB.IsSubscribed("c1"))
}

func TestChannelManagerGetChannelMissing(t *testing.T) {
	cm := NewChannelManager()
	_, ok := cm.GetChannel("room:missing")
	assert.False(t, ok)
}
