package observability

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestOtelLogCore_ForwardsEntries 验证 zapcore.Core 桥接将条目写入 channel。
func TestOtelLogCore_ForwardsEntries(t *testing.T) {
	ch := make(chan logEntry, 16)
	core := &otelLogCore{
		level: zapcore.DebugLevel,
		ch:    ch,
	}

	// Enabled/Check 过滤
	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello", Time: time.Now()}
	if !core.Enabled(ent.Level) {
		t.Fatal("info should be enabled at debug level")
	}

	ce := core.Check(ent, nil)
	if ce == nil {
		t.Fatal("Check() returned nil checked entry")
	}

	if err := core.Write(ent, []zapcore.Field{zap.String("k", "v")}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	select {
	case entry := <-ch:
		if entry.msg != "hello" || entry.level != zapcore.InfoLevel {
			t.Fatalf("unexpected entry: %+v", entry)
		}
		if len(entry.fields) != 1 {
			t.Fatalf("fields = %+v", entry.fields)
		}
	case <-time.After(time.Second):
		t.Fatal("entry not forwarded")
	}
}

// TestOtelLogCore_LevelFilter 验证级别过滤。
func TestOtelLogCore_LevelFilter(t *testing.T) {
	ch := make(chan logEntry, 16)
	core := &otelLogCore{level: zapcore.WarnLevel, ch: ch}

	if core.Enabled(zapcore.InfoLevel) {
		t.Fatal("info should be disabled at warn level")
	}
	if !core.Enabled(zapcore.ErrorLevel) {
		t.Fatal("error should be enabled at warn level")
	}

	// 低于级别的条目应被 Check 拦截
	ce := core.Check(zapcore.Entry{Level: zapcore.InfoLevel}, nil)
	if ce != nil {
		t.Fatal("Check() should return nil for below-threshold level")
	}
}

// TestOtelLogCore_WithClonesFields 验证 With 返回带继承字段的新 core。
func TestOtelLogCore_WithClonesFields(t *testing.T) {
	ch := make(chan logEntry, 16)
	core := &otelLogCore{level: zapcore.DebugLevel, ch: ch}

	cloned := core.With([]zapcore.Field{zap.String("svc", "jimu")})
	if core == cloned {
		t.Fatal("With() should return a new core")
	}

	_ = core.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "a"}, nil)
	_ = cloned.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "b"}, nil)

	first := <-ch
	second := <-ch
	if len(first.fields) != 0 {
		t.Fatalf("original core should not inherit fields, got %+v", first.fields)
	}
	if len(second.fields) != 1 || second.fields[0].Key != "svc" {
		t.Fatalf("cloned core fields = %+v", second.fields)
	}
}

// TestOtelLogCore_BufferFullDrops 验证缓冲满时丢弃而非阻塞。
func TestOtelLogCore_BufferFullDrops(t *testing.T) {
	ch := make(chan logEntry, 1)
	core := &otelLogCore{level: zapcore.DebugLevel, ch: ch}

	// 填满缓冲
	_ = core.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "full"}, nil)
	done := make(chan struct{})
	go func() {
		// 满时 Write 应立即返回（丢弃）
		for i := 0; i < 100; i++ {
			if err := core.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "drop"}, nil); err != nil {
				t.Errorf("Write() error = %v", err)
				return
			}
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Write() blocked when buffer full")
	}
	<-ch // 清掉缓冲中的一条
}
