package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Claims struct {
	UserID    uint64 `json:"user_id"`
	SessionID string `json:"sid"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret           []byte
	issuer           string
	accessExpireMin  time.Duration
	refreshExpireDay time.Duration
}

func New(secret, issuer string, accessExpireMin, refreshExpireDay int) *JWT {
	return &JWT{
		secret:           []byte(secret),
		issuer:           issuer,
		accessExpireMin:  time.Duration(accessExpireMin) * time.Minute,
		refreshExpireDay: time.Duration(refreshExpireDay) * 24 * time.Hour,
	}
}

func (j *JWT) GenerateAccess(userID uint64, sessionID string) (string, error) {
	claims := j.newClaims(userID, sessionID, TokenTypeAccess, j.accessExpireMin)
	return j.sign(claims)
}

func (j *JWT) GenerateRefresh(userID uint64, sessionID string) (string, Claims, error) {
	claims := j.newClaims(userID, sessionID, TokenTypeRefresh, j.refreshExpireDay)
	token, err := j.sign(claims)
	return token, claims, err
}

func (j *JWT) Parse(tokenString, expectedType string) (*Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(j.issuer),
		jwt.WithExpirationRequired(),
	)

	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, errors.New("invalid token type")
	}
	if claims.SessionID == "" || claims.ID == "" || claims.Subject == "" {
		return nil, errors.New("invalid token claims")
	}
	if claims.Issuer != j.issuer {
		return nil, errors.New("invalid issuer")
	}
	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return nil, err
	}
	if userID != claims.UserID {
		return nil, errors.New("invalid subject")
	}
	return claims, nil
}

func (j *JWT) newClaims(userID uint64, sessionID, tokenType string, ttl time.Duration) Claims {
	now := time.Now()
	return Claims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   strconv.FormatUint(userID, 10),
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
}

func (j *JWT) sign(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = "JWT"
	return token.SignedString(j.secret)
}
