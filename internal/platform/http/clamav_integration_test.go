package http

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// clamavAddr 测试用 clamd 地址，默认 127.0.0.1:3310，可由 CLAMAV_ADDR 覆盖。
func clamavAddr() string {
	if a := os.Getenv("CLAMAV_ADDR"); a != "" {
		return a
	}
	return "127.0.0.1:3310"
}

// skipUnlessClamAV 本地无 clamd 时跳过集成测试。
// 与 testutil.SkipUnlessMysql 同模式：TCP 探测，不可达即 skip。
func skipUnlessClamAV(t *testing.T) string {
	t.Helper()
	addr := clamavAddr()
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		t.Skipf("clamd %s unreachable; skipping clamav integration test (set CLAMAV_ADDR to override)", addr)
	}
	_ = conn.Close()
	return addr
}

// EICAR 标准病毒测试串（ClamAV 默认识别为 EICAR-Test-Signature）。
const eicarTestString = "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"

// TestClamAVScannerIntegrationClean 真实 clamd 扫描干净内容，应返回 clean=true。
func TestClamAVScannerIntegrationClean(t *testing.T) {
	addr := skipUnlessClamAV(t)
	s := NewClamAVScanner(ClamAVConfig{Address: addr, Timeout: 30 * time.Second})
	clean, err := s.Scan(context.Background(), strings.NewReader("hello world, this is clean content"))
	assertNoErr(t, err)
	if !clean {
		t.Fatal("expected clean file to scan as clean")
	}
}

// TestClamAVScannerIntegrationEICAR 真实 clamd 扫描 EICAR 测试串，应返回 clean=false（检测到威胁）。
func TestClamAVScannerIntegrationEICAR(t *testing.T) {
	addr := skipUnlessClamAV(t)
	s := NewClamAVScanner(ClamAVConfig{Address: addr, Timeout: 30 * time.Second})
	clean, err := s.Scan(context.Background(), strings.NewReader(eicarTestString))
	assertNoErr(t, err)
	if clean {
		t.Fatal("expected EICAR test string to be detected as infected")
	}
}

func assertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
