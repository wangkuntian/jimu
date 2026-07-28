package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint64 `json:"user_id"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret           []byte
	accessExpireMin  time.Duration
	refreshExpireDay time.Duration
}

func New(secret string, accessExpireMin, refreshExpireDay int) *JWT {
	return &JWT{
		secret:           []byte(secret),
		accessExpireMin:  time.Duration(accessExpireMin) * time.Minute,
		refreshExpireDay: time.Duration(refreshExpireDay) * 24 * time.Hour,
	}
}

func (j *JWT) Generate(userID uint64) (string, string, error) {
	accessClaims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessExpireMin)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString(j.secret)
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshExpireDay)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshString, err := refreshToken.SignedString(j.secret)
	if err != nil {
		return "", "", err
	}

	return accessString, refreshString, nil
}

func (j *JWT) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
