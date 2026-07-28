package config

import (
	"os"
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
