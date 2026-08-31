package observability

import (
	"encoding/base64"
	"testing"
)

func TestOtlpHeaders_DefaultOrgNoAuth(t *testing.T) {
	headers := otlpHeaders(TracingConfig{})
	if headers["organization"] != "default" {
		t.Fatalf("organization = %q, want default", headers["organization"])
	}
	if _, ok := headers["authorization"]; ok {
		t.Fatal("authorization should be absent when AuthEmail empty")
	}
}

func TestOtlpHeaders_BasicAuth(t *testing.T) {
	headers := otlpHeaders(TracingConfig{
		AuthEmail:    "admin@jimu.local",
		AuthPassword: "admin",
		OrgID:        "prod",
	})
	if headers["organization"] != "prod" {
		t.Fatalf("organization = %q, want prod", headers["organization"])
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin@jimu.local:admin"))
	if headers["authorization"] != want {
		t.Fatalf("authorization = %q, want %q", headers["authorization"], want)
	}
}

func TestOtlpHeaders_ExplicitDefaultOrg(t *testing.T) {
	headers := otlpHeaders(TracingConfig{OrgID: "", AuthEmail: "u", AuthPassword: "p"})
	if headers["organization"] != "default" {
		t.Fatalf("empty OrgID should fall back to default, got %q", headers["organization"])
	}
	if headers["authorization"] == "" {
		t.Fatal("authorization should be set when AuthEmail provided")
	}
}
