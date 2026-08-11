// Package id 提供分布式 ID 生成器。
// 雪花 ID 适合需要全局有序、按时间排序的场景（如消息聚合、跨库分片）。
// 普通场景保持 AUTO_INCREMENT 主键即可，无需引入本包。
package id

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Generator 生成全局唯一 ID
type Generator interface {
	// NextID 返回下一个唯一 ID。时钟回拨等不可恢复错误返回 error。
	NextID() (uint64, error)
}

const (
	// 雪花布局：1bit 符号 + 41bit 时间戳(ms) + 10bit worker + 12bit 序列
	workerIDBits   = 10
	sequenceBits   = 12
	maxWorkerID    = -1 ^ (-1 << workerIDBits) // 1023
	sequenceMask   = -1 ^ (-1 << sequenceBits) // 4095
	workerIDShift  = sequenceBits
	timestampShift = workerIDBits + sequenceBits
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

// ErrWorkerIDOutOfRange worker ID 超出 0-1023 范围
var ErrWorkerIDOutOfRange = errors.New("id: worker id out of range (0-1023)")

// ErrClockBackwards 检测到时钟回拨
var ErrClockBackwards = errors.New("id: clock moved backwards")

// snowflake 雪花 ID 生成器
type snowflake struct {
	mu        sync.Mutex
	workerID  int64
	sequence  int64
	lastStamp int64
}

// NewSnowflake 创建雪花 ID 生成器。workerID 范围 0-1023。
func NewSnowflake(workerID int64) (Generator, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, ErrWorkerIDOutOfRange
	}
	return &snowflake{workerID: workerID}, nil
}

func (s *snowflake) NextID() (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stamp := time.Now().UnixMilli() - epoch
	if stamp < s.lastStamp {
		return 0, ErrClockBackwards
	}
	if stamp == s.lastStamp {
		s.sequence = (s.sequence + 1) & sequenceMask
		// 序列耗尽，等待下一毫秒
		if s.sequence == 0 {
			for stamp <= s.lastStamp {
				stamp = time.Now().UnixMilli() - epoch
			}
		}
	} else {
		s.sequence = 0
	}
	s.lastStamp = stamp

	id := (uint64(stamp) << timestampShift) |
		(uint64(s.workerID) << workerIDShift) |
		uint64(s.sequence)
	return id, nil
}

// uuidGenerator 基于 UUID 的生成器，适合客户端生成或无需排序的场景
type uuidGenerator struct{}

// NewUUIDGenerator 创建 UUID 生成器（取 UUID 的 uint64 哈希）
func NewUUIDGenerator() Generator {
	return &uuidGenerator{}
}

func (g *uuidGenerator) NextID() (uint64, error) {
	// UUID 128bit，取低 64bit 保证唯一性概率足够（碰撞率 ~2^-64）
	id := uuid.New()
	return binaryID(id), nil
}

func binaryID(u uuid.UUID) uint64 {
	// UUID 128bit，取后 64bit 作为 ID
	return binary.BigEndian.Uint64(u[8:])
}
