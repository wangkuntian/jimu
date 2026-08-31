package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	redistore "jimu/internal/platform/redis"

	"github.com/redis/go-redis/v9"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrTokenReuse      = errors.New("refresh token reuse detected")
)

type SessionStore interface {
	Create(ctx context.Context, userID uint64, sessionID, tokenID string, ttl time.Duration) error
	Rotate(ctx context.Context, userID uint64, sessionID, oldTokenID, newTokenID string, ttl time.Duration) error
	Revoke(ctx context.Context, userID uint64, sessionID string) error
	RevokeAll(ctx context.Context, userID uint64) error
}

type RedisSessionStore struct {
	client sessionRedis
}

func NewRedisSessionStore(client redistore.Client) SessionStore {
	return newRedisSessionStore(redisClientAdapter{Client: client})
}

func newRedisSessionStore(client sessionRedis) SessionStore {
	return &RedisSessionStore{client: client}
}

type sessionRedis interface {
	redis.Scripter
	TxPipeline() sessionPipeline
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
}

type sessionPipeline interface {
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exec(ctx context.Context) ([]redis.Cmder, error)
}

type redisClientAdapter struct {
	redistore.Client
}

func (a redisClientAdapter) TxPipeline() sessionPipeline {
	return a.Client.TxPipeline()
}

func (s *RedisSessionStore) Create(ctx context.Context, userID uint64, sessionID, tokenID string, ttl time.Duration) error {
	key := sessionKey(sessionID)
	userKey := userSessionsKey(userID)
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key, "user_id", strconv.FormatUint(userID, 10), "token_id", tokenID)
	pipe.Expire(ctx, key, ttl)
	pipe.SAdd(ctx, userKey, sessionID)
	pipe.Expire(ctx, userKey, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisSessionStore) Rotate(ctx context.Context, userID uint64, sessionID, oldTokenID, newTokenID string, ttl time.Duration) error {
	script := redis.NewScript(`
local key = KEYS[1]
local userKey = KEYS[2]
local expectedUserID = ARGV[1]
local oldTokenID = ARGV[2]
local newTokenID = ARGV[3]
local ttl = tonumber(ARGV[4])
local userID = redis.call("HGET", key, "user_id")
if not userID then
  return 0
end
if userID ~= expectedUserID then
  return 0
end
local current = redis.call("HGET", key, "token_id")
if not current then
  return 0
end
if current ~= oldTokenID then
  return -1
end
redis.call("HSET", key, "token_id", newTokenID)
redis.call("EXPIRE", key, ttl)
redis.call("EXPIRE", userKey, ttl)
return 1
`)
	res, err := script.Run(ctx, s.client, []string{sessionKey(sessionID), userSessionsKey(userID)}, strconv.FormatUint(userID, 10), oldTokenID, newTokenID, int(ttl.Seconds())).Int()
	if err != nil {
		return err
	}
	switch res {
	case 1:
		return nil
	case 0:
		return ErrSessionNotFound
	case -1:
		return ErrTokenReuse
	default:
		return fmt.Errorf("unexpected rotate result: %d", res)
	}
}

func (s *RedisSessionStore) Revoke(ctx context.Context, userID uint64, sessionID string) error {
	key := sessionKey(sessionID)
	userKey := userSessionsKey(userID)
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, userKey, sessionID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisSessionStore) RevokeAll(ctx context.Context, userID uint64) error {
	userKey := userSessionsKey(userID)
	sessionIDs, err := s.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	for _, sessionID := range sessionIDs {
		pipe.Del(ctx, sessionKey(sessionID))
	}
	pipe.Del(ctx, userKey)
	_, err = pipe.Exec(ctx)
	return err
}

func sessionKey(sessionID string) string {
	return "jimu:auth:session:" + sessionID
}

func userSessionsKey(userID uint64) string {
	return fmt.Sprintf("jimu:auth:user:%d:sessions", userID)
}
