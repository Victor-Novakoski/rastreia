// Package auth issues and checks JWTs and guards routes by role.
package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	RoleAdmin  = "admin"
	RoleDriver = "driver"
)

// Claims is what the API knows about the caller after authentication.
type Claims struct {
	UserID int64
	Role   string
}

type tokenClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokens(secret string, ttl time.Duration) *Tokens {
	return &Tokens{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (t *Tokens) Issue(c Claims) (string, error) {
	now := t.now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		Role: c.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(c.UserID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	})
	return token.SignedString(t.secret)
}

func (t *Tokens) Parse(raw string) (Claims, error) {
	var tc tokenClaims
	_, err := jwt.ParseWithClaims(raw, &tc, func(*jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithTimeFunc(t.now))
	if err != nil {
		return Claims{}, err
	}
	id, err := strconv.ParseInt(tc.Subject, 10, 64)
	if err != nil {
		return Claims{}, fmt.Errorf("invalid subject: %w", err)
	}
	if tc.Role != RoleAdmin && tc.Role != RoleDriver {
		return Claims{}, errors.New("invalid role")
	}
	return Claims{UserID: id, Role: tc.Role}, nil
}
