// Package auth handles password hashing and JWT issuing/verification for
// voxa-api. Auth is independent per-app (not shared with other pixelio
// products) — see the project's own users table.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidToken = errors.New("invalid token")

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type claims struct {
	jwt.RegisteredClaims
}

func IssueToken(secret string, ttl time.Duration, userID string) (string, error) {
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

// VerifyToken returns the user id (subject) encoded in a valid token.
func VerifyToken(secret, tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}
	c, ok := token.Claims.(*claims)
	if !ok || c.Subject == "" {
		return "", ErrInvalidToken
	}
	return c.Subject, nil
}
