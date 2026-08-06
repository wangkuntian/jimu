package feature

import (
	"context"
	"sync"
)

// Flag 特性开关
type Flag struct {
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`      // 是否启用
	Percentage  int               `json:"percentage"`   // 灰度百分比 (0-100)
	Users       map[string]bool   `json:"-"`            // 白名单用户
	Metadata    map[string]string `json:"metadata"`     // 元数据
}

// IsEnabled 判断特性是否对指定用户启用
func (f *Flag) IsEnabled(userID string) bool {
	if !f.Enabled {
		return false
	}
	// 白名单用户始终启用
	if f.Users != nil {
		if enabled, ok := f.Users[userID]; ok {
			return enabled
		}
	}
	// 灰度百分比
	if f.Percentage >= 100 {
		return true
	}
	if f.Percentage <= 0 {
		return false
	}
	// 基于用户 ID 的哈希决定
	if userID != "" {
		return hashUserID(userID)%100 < f.Percentage
	}
	return false
}

// Manager 特性开关管理器
type Manager struct {
	mu    sync.RWMutex
	flags map[string]*Flag
}

// NewManager 创建特性开关管理器
func NewManager() *Manager {
	return &Manager{
		flags: make(map[string]*Flag),
	}
}

// Register 注册特性开关
func (m *Manager) Register(flag Flag) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flags[flag.Name] = &flag
}

// IsEnabled 判断特性是否启用
func (m *Manager) IsEnabled(flagName string, userID ...string) bool {
	m.mu.RLock()
	flag, ok := m.flags[flagName]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	uid := ""
	if len(userID) > 0 {
		uid = userID[0]
	}
	return flag.IsEnabled(uid)
}

// Get 获取特性开关
func (m *Manager) Get(flagName string) (*Flag, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	flag, ok := m.flags[flagName]
	return flag, ok
}

// List 列出所有特性开关
func (m *Manager) List() []Flag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Flag, 0, len(m.flags))
	for _, f := range m.flags {
		result = append(result, *f)
	}
	return result
}

// Update 更新特性开关
func (m *Manager) Update(flagName string, updates func(*Flag)) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	flag, ok := m.flags[flagName]
	if !ok {
		return false
	}
	updates(flag)
	return true
}

// hashUserID 简单的用户 ID 哈希
func hashUserID(userID string) int {
	h := 0
	for _, c := range userID {
		h = 31*h + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// key 用于在 context 中存储 Manager
type contextKey struct{}

var managerKey = contextKey{}

// WithManager 将 Manager 注入 context
func WithManager(ctx context.Context, m *Manager) context.Context {
	return context.WithValue(ctx, managerKey, m)
}

// FromContext 从 context 获取 Manager
func FromContext(ctx context.Context) (*Manager, bool) {
	m, ok := ctx.Value(managerKey).(*Manager)
	return m, ok
}
