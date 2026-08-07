package ws

import (
	"sync"
)

// Channel 表示一个消息频道（用户频道或房间频道）
type Channel struct {
	Name        string          // user:123 / room:abc / broadcast
	Type        string          // user / room / broadcast
	Subscribers map[string]bool // connectionID -> subscribed
	mu          sync.RWMutex
}

// NewChannel 创建频道
func NewChannel(name, channelType string) *Channel {
	return &Channel{
		Name:        name,
		Type:        channelType,
		Subscribers: make(map[string]bool),
	}
}

// Subscribe 订阅频道
func (c *Channel) Subscribe(connID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Subscribers[connID] = true
}

// Unsubscribe 取消订阅
func (c *Channel) Unsubscribe(connID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Subscribers, connID)
}

// SubscriberCount 返回订阅者数量
func (c *Channel) SubscriberCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Subscribers)
}

// IsSubscribed 检查连接是否已订阅
func (c *Channel) IsSubscribed(connID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Subscribers[connID]
}

// ChannelManager 频道管理器
type ChannelManager struct {
	mu       sync.RWMutex
	channels map[string]*Channel
}

// NewChannelManager 创建频道管理器
func NewChannelManager() *ChannelManager {
	cm := &ChannelManager{
		channels: make(map[string]*Channel),
	}
	// 预创建广播频道
	cm.channels[ChannelBroadcast] = NewChannel(ChannelBroadcast, "broadcast")
	return cm
}

// Subscribe 订阅频道（不存在则自动创建）
func (cm *ChannelManager) Subscribe(connID, channelName string) {
	cm.mu.Lock()
	ch, ok := cm.channels[channelName]
	if !ok {
		chType := "user"
		if len(channelName) > len(ChannelRoomPrefix) && channelName[:len(ChannelRoomPrefix)] == ChannelRoomPrefix {
			chType = "room"
		} else if channelName == ChannelBroadcast {
			chType = "broadcast"
		}
		ch = NewChannel(channelName, chType)
		cm.channels[channelName] = ch
	}
	cm.mu.Unlock()

	ch.Subscribe(connID)
}

// Unsubscribe 取消订阅频道
func (cm *ChannelManager) Unsubscribe(connID, channelName string) {
	cm.mu.RLock()
	ch, ok := cm.channels[channelName]
	cm.mu.RUnlock()
	if ok {
		ch.Unsubscribe(connID)
	}
}

// UnsubscribeAll 取消连接在所有频道的订阅
func (cm *ChannelManager) UnsubscribeAll(connID string, channels []string) {
	for _, name := range channels {
		cm.Unsubscribe(connID, name)
	}
}

// GetSubscribers 获取频道的所有订阅者
func (cm *ChannelManager) GetSubscribers(channelName string) []string {
	cm.mu.RLock()
	ch, ok := cm.channels[channelName]
	cm.mu.RUnlock()
	if !ok {
		return nil
	}
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	subs := make([]string, 0, len(ch.Subscribers))
	for id := range ch.Subscribers {
		subs = append(subs, id)
	}
	return subs
}

// GetChannel 获取频道信息
func (cm *ChannelManager) GetChannel(channelName string) (*Channel, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	ch, ok := cm.channels[channelName]
	return ch, ok
}

// ChannelCount 返回频道总数
func (cm *ChannelManager) ChannelCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.channels)
}
