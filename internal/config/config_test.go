package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.HTTP.Port == 0 {
		t.Error("expected HTTP.Port to be set")
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("JIMU__HTTP__PORT", "9999")
	defer os.Unsetenv("JIMU__HTTP__PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.HTTP.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.HTTP.Port)
	}
}

func TestValidateHTTPMode(t *testing.T) {
	os.Setenv("JIMU__HTTP__MODE", "invalid")
	defer os.Unsetenv("JIMU__HTTP__MODE")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid http.mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid http.mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLogLevel(t *testing.T) {
	os.Setenv("JIMU__LOG__LEVEL", "trace")
	defer os.Unsetenv("JIMU__LOG__LEVEL")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid log.level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log.level") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLogFormat(t *testing.T) {
	os.Setenv("JIMU__LOG__FORMAT", "xml")
	defer os.Unsetenv("JIMU__LOG__FORMAT")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid log.format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log.format") {
		t.Errorf("unexpected error: %v", err)
	}
}
