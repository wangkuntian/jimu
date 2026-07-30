package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseRejectsWrongTokenType(t *testing.T) {
	j := New(strings.Repeat("s", 32), "jimu", 30, 7)
	refresh, _, err := j.GenerateRefresh(42, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Parse(refresh, TokenTypeAccess); err == nil {
		t.Fatal("refresh token accepted as access token")
	}
}

func TestJWTPopulatesTypedClaims(t *testing.T) {
	j := New(strings.Repeat("s", 32), "jimu", 30, 7)
	access, err := j.GenerateAccess(42, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	refresh, refreshClaims, err := j.GenerateRefresh(42, "session-1")
	if err != nil {
		t.Fatal(err)
	}

	accessClaims, err := j.Parse(access, TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if accessClaims.TokenType != TokenTypeAccess || accessClaims.SessionID != "session-1" {
		t.Fatalf("access claims = %#v", accessClaims)
	}
	if accessClaims.UserID != 42 || accessClaims.Subject != "42" {
		t.Fatalf("subject/user mismatch: %#v", accessClaims)
	}
	if accessClaims.ID == "" || accessClaims.Issuer != "jimu" {
		t.Fatalf("access claims missing metadata: %#v", accessClaims)
	}
	if accessClaims.ExpiresAt == nil || time.Until(accessClaims.ExpiresAt.Time) <= 0 {
		t.Fatalf("access expiry invalid: %#v", accessClaims)
	}

	parsedRefresh, err := j.Parse(refresh, TokenTypeRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if parsedRefresh.TokenType != TokenTypeRefresh || parsedRefresh.SessionID != "session-1" {
		t.Fatalf("refresh claims = %#v", parsedRefresh)
	}
	if parsedRefresh.ID == "" || parsedRefresh.Issuer != "jimu" {
		t.Fatalf("refresh claims missing metadata: %#v", parsedRefresh)
	}
	if refreshClaims.TokenType != TokenTypeRefresh || refreshClaims.Subject != "42" {
		t.Fatalf("generated refresh claims = %#v", refreshClaims)
	}

	token, _, err := new(jwt.Parser).ParseUnverified(access, &Claims{})
	if err != nil {
		t.Fatal(err)
	}
	if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		t.Fatalf("alg = %s, want %s", token.Method.Alg(), jwt.SigningMethodHS256.Alg())
	}
}
