package authmemory

import (
	"context"
	"sync"
	"time"

	"jimu/internal/platform/identity"
)

type session struct {
	userID  uint64
	tokenID string
	expires time.Time
}

// SessionStore 是进程内会话存储。只在单实例内有效，重启即丢失。
type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]session
	users    map[uint64]map[string]struct{}
}

// NewSessionStore 创建进程内会话存储。
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: map[string]session{},
		users:    map[uint64]map[string]struct{}{},
	}
}

// Create 实现 identity.SessionStore。
func (s *SessionStore) Create(_ context.Context, userID uint64, sessionID, tokenID string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session{userID: userID, tokenID: tokenID, expires: time.Now().Add(ttl)}
	if s.users[userID] == nil {
		s.users[userID] = map[string]struct{}{}
	}
	s.users[userID][sessionID] = struct{}{}
	return nil
}

// Rotate 实现 identity.SessionStore，并检测 refresh token 复用。
func (s *SessionStore) Rotate(_ context.Context, userID uint64, sessionID, oldTokenID, newTokenID string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[sessionID]
	if !ok || time.Now().After(current.expires) || current.userID != userID {
		s.deleteLocked(userID, sessionID)
		return identity.ErrSessionNotFound
	}
	if current.tokenID != oldTokenID {
		return identity.ErrTokenReuse
	}
	current.tokenID = newTokenID
	current.expires = time.Now().Add(ttl)
	s.sessions[sessionID] = current
	return nil
}

// Revoke 实现 identity.SessionStore。
func (s *SessionStore) Revoke(_ context.Context, userID uint64, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteLocked(userID, sessionID)
	return nil
}

// RevokeAll 实现 identity.SessionStore。
func (s *SessionStore) RevokeAll(_ context.Context, userID uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sessionID := range s.users[userID] {
		delete(s.sessions, sessionID)
	}
	delete(s.users, userID)
	return nil
}

func (s *SessionStore) deleteLocked(userID uint64, sessionID string) {
	delete(s.sessions, sessionID)
	if sessions := s.users[userID]; sessions != nil {
		delete(sessions, sessionID)
		if len(sessions) == 0 {
			delete(s.users, userID)
		}
	}
}
