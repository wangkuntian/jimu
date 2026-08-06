package application

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9"
)

// SystemStatus 和 MemoryStats 定义在 monitoring_service.go 中

// OnlineUser 在线用户信息
type OnlineUser struct {
	UserID    uint64    `json:"user_id"`
	Username  string    `json:"username"`
	SessionID string    `json:"session_id"`
	LoginAt   time.Time `json:"login_at"`
}

// Service 管理端服务
type Service struct {
	startTime time.Time
	version   string
	env       string
	redis     *redis.Client
}

// NewService 创建管理端服务
func NewService(version, env string, rdb *redis.Client) *Service {
	return &Service{
		startTime: time.Now(),
		version:   version,
		env:       env,
		redis:     rdb,
	}
}

// Version 返回版本号
func (s *Service) Version() string { return s.version }

// Env 返回运行环境
func (s *Service) Env() string { return s.env }

// GetStatus 获取系统状态
func (s *Service) GetStatus() SystemStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := SystemStatus{
		Version:      s.version,
		Environment:  s.env,
		StartTime:    s.startTime,
		Uptime:       time.Since(s.startTime).String(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		Memory: MemoryStats{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
			HeapAlloc:  memStats.HeapAlloc,
			HeapInuse:  memStats.HeapInuse,
		},
	}

	// 检查 Redis 连接
	if s.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		status.RedisConnected = s.redis.Ping(ctx).Err() == nil
	}

	return status
}

// GetOnlineUsers 获取在线用户列表（从 Redis 读取）
func (s *Service) GetOnlineUsers(ctx context.Context) ([]OnlineUser, error) {
	if s.redis == nil {
		return []OnlineUser{}, nil
	}

	// 扫描所有用户的会话
	var users []OnlineUser
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = s.redis.Scan(ctx, cursor, "jimu:auth:user:*:sessions", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan sessions: %w", err)
		}

		for _, key := range keys {
			// 提取 user_id: jimu:auth:user:{userID}:sessions
			userID, sessionIDs, err := s.getUserSessions(ctx, key)
			if err != nil {
				continue
			}
			for _, sid := range sessionIDs {
				users = append(users, OnlineUser{
					UserID:    userID,
					SessionID: sid,
				})
			}
		}

		if cursor == 0 {
			break
		}
	}

	return users, nil
}

// ForceLogout 强制用户下线（删除所有会话）
func (s *Service) ForceLogout(ctx context.Context, userID uint64) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("jimu:auth:user:%d:sessions", userID)

	// 获取所有会话
	sessionIDs, err := s.redis.SMembers(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("get sessions: %w", err)
	}

	// 删除所有会话数据和用户会话集合
	pipe := s.redis.Pipeline()
	for _, sid := range sessionIDs {
		pipe.Del(ctx, fmt.Sprintf("jimu:auth:session:%s", sid))
	}
	pipe.Del(ctx, key)
	_, err = pipe.Exec(ctx)

	return err
}

// ForceLogoutSession 强制指定会话下线
func (s *Service) ForceLogoutSession(ctx context.Context, userID uint64, sessionID string) error {
	if s.redis == nil {
		return nil
	}

	pipe := s.redis.Pipeline()
	pipe.Del(ctx, fmt.Sprintf("jimu:auth:session:%s", sessionID))
	pipe.SRem(ctx, fmt.Sprintf("jimu:auth:user:%d:sessions", userID), sessionID)
	_, err := pipe.Exec(ctx)

	return err
}

// getUserSessions 从 Redis 获取用户会话
func (s *Service) getUserSessions(ctx context.Context, key string) (uint64, []string, error) {
	var userID uint64
	_, err := fmt.Sscanf(key, "jimu:auth:user:%d:sessions", &userID)
	if err != nil {
		return 0, nil, err
	}
	sessionIDs, err := s.redis.SMembers(ctx, key).Result()
	if err != nil {
		return 0, nil, err
	}
	return userID, sessionIDs, nil
}
