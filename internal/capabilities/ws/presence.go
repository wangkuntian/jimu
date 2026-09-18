package ws

import (
	"sync"
	"time"
)

// 在线状态
const (
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusTyping  = "typing"
)

// HeartbeatInterval 心跳间隔（默认 30 秒）
const HeartbeatInterval = 30 * time.Second

// MaxMissedHeartbeats 最大允许丢失心跳次数
const MaxMissedHeartbeats = 3

// TypingTimeout 输入状态超时（默认 3 秒）
const TypingTimeout = 3 * time.Second

// Presence 用户在线状态
type Presence struct {
	UserID        uint64
	Status        string
	LastHeartbeat time.Time
	LastTyping    time.Time
	MissedBeats   int
	ConnectedAt   time.Time
	ConnID        string
}

// IsOnline 检查是否在线
func (p *Presence) IsOnline() bool {
	return p.Status != StatusOffline
}

// IsTyping 检查是否正在输入
func (p *Presence) IsTyping() bool {
	if p.Status == StatusOffline {
		return false
	}
	return time.Since(p.LastTyping) < TypingTimeout
}

// Stale 检查心跳是否超时
func (p *Presence) Stale() bool {
	return time.Since(p.LastHeartbeat) > HeartbeatInterval*time.Duration(MaxMissedHeartbeats)
}

// PresenceManager 在线状态管理器
type PresenceManager struct {
	mu        sync.RWMutex
	presences map[uint64]*Presence
}

// NewPresenceManager 创建在线状态管理器
func NewPresenceManager() *PresenceManager {
	return &PresenceManager{
		presences: make(map[uint64]*Presence),
	}
}

// Online 用户上线
func (pm *PresenceManager) Online(userID uint64, connID string) *Presence {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	now := time.Now()
	p := &Presence{
		UserID:        userID,
		Status:        StatusOnline,
		LastHeartbeat: now,
		ConnectedAt:   now,
		ConnID:        connID,
		MissedBeats:   0,
	}
	pm.presences[userID] = p
	return p
}

// Offline 用户下线
func (pm *PresenceManager) Offline(userID uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if p, ok := pm.presences[userID]; ok {
		p.Status = StatusOffline
		delete(pm.presences, userID)
	}
}

// Heartbeat 更新心跳
func (pm *PresenceManager) Heartbeat(userID uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if p, ok := pm.presences[userID]; ok {
		p.LastHeartbeat = time.Now()
		p.MissedBeats = 0
		if p.Status != StatusOnline {
			p.Status = StatusOnline
		}
	}
}

// SetTyping 设置用户输入状态
func (pm *PresenceManager) SetTyping(userID uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if p, ok := pm.presences[userID]; ok {
		p.LastTyping = time.Now()
	}
}

// GetPresence 获取用户在线状态
func (pm *PresenceManager) GetPresence(userID uint64) (*Presence, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	p, ok := pm.presences[userID]
	return p, ok
}

// IsOnline 检查用户是否在线
func (pm *PresenceManager) IsOnline(userID uint64) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	p, ok := pm.presences[userID]
	return ok && p.IsOnline()
}

// OnlineCount 返回在线用户数
func (pm *PresenceManager) OnlineCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	count := 0
	for _, p := range pm.presences {
		if p.IsOnline() {
			count++
		}
	}
	return count
}

// OnlineUsers 返回所有在线用户 ID
func (pm *PresenceManager) OnlineUsers() []uint64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	users := make([]uint64, 0)
	for id, p := range pm.presences {
		if p.IsOnline() {
			users = append(users, id)
		}
	}
	return users
}

// StaleUsers 返回心跳超时的用户 ID
func (pm *PresenceManager) StaleUsers() []uint64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	var users []uint64
	for id, p := range pm.presences {
		if p.IsOnline() && p.Stale() {
			users = append(users, id)
		}
	}
	return users
}

// AllPresences 返回所有在线状态（用于接口查询）
func (pm *PresenceManager) AllPresences() []*Presence {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make([]*Presence, 0, len(pm.presences))
	for _, p := range pm.presences {
		result = append(result, p)
	}
	return result
}
