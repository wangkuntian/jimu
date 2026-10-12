package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// testJWT 返回与生产 New 同构、字段齐全的实例（HS256 + issuer=jimu）。
func testJWT() *JWT {
	return New(strings.Repeat("k", 32), "jimu", 30, 7)
}

// validClaims 构造一份字段齐全、可被 Parse 接受的 access claims，供篡改/过期等反向场景复用。
func validClaims(issuer string, exp time.Time) Claims {
	return Claims{
		UserID:    42,
		TenantID:  5,
		SessionID: "sid-1",
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "42",
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
}

// A1.1 有效 HS256 token：通过验签，且能解析出 sub 与 tid。
func TestA1_1_ValidTokenParsesSubjectAndTenant(t *testing.T) {
	j := testJWT()
	raw, err := j.GenerateAccess(42, 5, "sid-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := j.Parse(raw, TokenTypeAccess)
	if err != nil {
		t.Fatalf("有效 HS256 token 必须通过：%v", err)
	}
	if claims.Subject != "42" || claims.UserID != 42 {
		t.Fatalf("sub/user_id = %q/%d, want \"42\"/42", claims.Subject, claims.UserID)
	}
	if claims.TenantID != 5 {
		t.Fatalf("tid = %d, want 5", claims.TenantID)
	}
}

// A1.2 过期 token 必须被拒绝（Parse 走 WithExpirationRequired）。
func TestA1_2_ExpiredTokenIsRejected(t *testing.T) {
	j := testJWT()
	raw, err := j.sign(validClaims(j.issuer, time.Now().Add(-time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Parse(raw, TokenTypeAccess); err == nil {
		t.Fatal("过期 token 必须被拒绝")
	}
}

// A1.3 签名被篡改（换密钥签名 / 直接改签名字节）必须被拒绝。
func TestA1_3_TamperedSignatureIsRejected(t *testing.T) {
	j := testJWT()
	raw, err := j.GenerateAccess(42, 5, "sid-1")
	if err != nil {
		t.Fatal(err)
	}

	other := New(strings.Repeat("o", 32), "jimu", 30, 7)
	forged, err := other.GenerateAccess(42, 5, "sid-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Parse(forged, TokenTypeAccess); err == nil {
		t.Fatal("异密钥签名的 token 必须被拒绝")
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("token 结构异常：%q", raw)
	}
	sig := []byte(parts[2])
	if sig[0] == 'A' {
		sig[0] = 'B'
	} else {
		sig[0] = 'A'
	}
	if _, err := j.Parse(parts[0]+"."+parts[1]+"."+string(sig), TokenTypeAccess); err == nil {
		t.Fatal("被篡改的签名字节必须被拒绝")
	}
}

// A1.4 alg=none token 必须被拒绝（jwt.go:70 的 WithValidMethods 白名单）。
func TestA1_4_NoneAlgTokenIsRejected(t *testing.T) {
	j := testJWT()
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims(j.issuer, time.Now().Add(time.Minute)))
	raw, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("构造 alg=none token 失败：%v", err)
	}
	if _, err := j.Parse(raw, TokenTypeAccess); err == nil {
		t.Fatal("alg=none token 必须被拒绝")
	}
}

// A1.5 非 HS256 算法一律拒绝：RSA 算法混淆撞 WithValidMethods + HMAC 类型校验两道防线；
// HS512 是合法 HMAC（能过类型校验）但不在白名单，因此也只能被 WithValidMethods 拦住。
func TestA1_5_NonHS256TokensAreRejected(t *testing.T) {
	j := testJWT()

	t.Run("RSA 算法混淆", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims(j.issuer, time.Now().Add(time.Minute))).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := j.Parse(raw, TokenTypeAccess); err == nil {
			t.Fatal("RSA 签名的 token 必须被拒绝")
		}
	})

	t.Run("HS512 不在白名单", func(t *testing.T) {
		raw, err := jwt.NewWithClaims(jwt.SigningMethodHS512, validClaims(j.issuer, time.Now().Add(time.Minute))).
			SignedString([]byte(strings.Repeat("k", 32)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := j.Parse(raw, TokenTypeAccess); err == nil {
			t.Fatal("HS512 必须被拒绝（只允许 HS256）")
		}
	})
}
