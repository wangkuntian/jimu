package http

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Scanner 文件安全扫描接口。返回 clean=true 表示无威胁，err 非 nil 表示扫描不可达/失败。
// 实现须 fail-closed：不可达时返回 (false, err)，调用方据此拒绝落库。
type Scanner interface {
	Scan(ctx context.Context, r io.Reader) (clean bool, err error)
}

// ClamAVConfig ClamAV 扫描器配置
type ClamAVConfig struct {
	Address   string        // clamd 监听地址，如 "127.0.0.1:3310"
	Timeout   time.Duration // 扫描超时（含连接），0 用默认 10s
	ChunkSize int           // INSTREAM 分块大小，0 用默认 64KB
}

// ClamAVScanner 基于 ClamAV INSTREAM 协议的扫描器。
// 仅用 stdlib net，无外部依赖（宪法 II）。
type ClamAVScanner struct {
	address   string
	timeout   time.Duration
	chunkSize int
}

// dialFunc 拨号函数，可替换以便测试注入管道连接。
var dialFunc = func(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
}

// NewClamAVScanner 创建 ClamAV 扫描器
func NewClamAVScanner(cfg ClamAVConfig) *ClamAVScanner {
	s := &ClamAVScanner{
		address:   cfg.Address,
		chunkSize: cfg.ChunkSize,
	}
	if s.address == "" {
		s.address = "127.0.0.1:3310"
	}
	if cfg.Timeout > 0 {
		s.timeout = cfg.Timeout
	} else {
		s.timeout = 10 * time.Second
	}
	if s.chunkSize <= 0 {
		s.chunkSize = 64 * 1024
	}
	return s
}

// Scan 通过 INSTREAM 协议流式扫描。
// 协议：连接后发 "zINSTREAM\0"，随后发送 [4字节大端长度][payload] 帧，
// 0 长度帧结束；clamd 返回 "stream: OK"（干净）或 "stream: FOUND <name>"（感染）。
// 任何连接/IO 错误返回 (false, err)，调用方据此 fail-closed 拒绝落库。
func (s *ClamAVScanner) Scan(ctx context.Context, r io.Reader) (bool, error) {
	conn, err := dialFunc("tcp", s.address, s.timeout)
	if err != nil {
		return false, fmt.Errorf("clamav dial: %w", err)
	}
	defer conn.Close()

	// 整体扫描受 deadline 约束，防慢速上传拖垮连接
	if err := conn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
		return false, fmt.Errorf("clamav deadline: %w", err)
	}

	// 发送 INSTREAM 命令
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return false, fmt.Errorf("clamav command: %w", err)
	}

	// 流式分块发送
	buf := make([]byte, s.chunkSize)
	for {
		n, rerr := r.Read(buf)
		if n > 0 {
			if err := binary.Write(conn, binary.BigEndian, uint32(n)); err != nil {
				return false, fmt.Errorf("clamav write chunk size: %w", err)
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return false, fmt.Errorf("clamav write chunk: %w", err)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return false, fmt.Errorf("clamav read input: %w", rerr)
		}
	}

	// 0 长度帧结束流
	if err := binary.Write(conn, binary.BigEndian, uint32(0)); err != nil {
		return false, fmt.Errorf("clamav write terminator: %w", err)
	}

	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil && resp == "" {
		return false, fmt.Errorf("clamav read response: %w", err)
	}
	resp = strings.TrimSpace(resp)
	switch {
	case strings.Contains(resp, "OK"):
		return true, nil
	case strings.Contains(resp, "FOUND"):
		return false, nil // 检测到威胁，非错误
	default:
		return false, fmt.Errorf("clamav unexpected response: %s", resp)
	}
}
