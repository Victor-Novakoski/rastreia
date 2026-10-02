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

	issuer   = "rastreia"
	audience = "rastreia-api"
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
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   strconv.FormatInt(c.UserID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	})
	return token.SignedString(t.secret)
}

func (t *Tokens) Parse(raw string) (Claims, error) {
	c, _, err := t.ParseExpiry(raw)
	return c, err
}

// ParseExpiry is Parse that also returns when the token expires, for
// connections that outlive a single request.
func (t *Tokens) ParseExpiry(raw string) (Claims, time.Time, error) {
	var tc tokenClaims
	_, err := jwt.ParseWithClaims(raw, &tc, func(*jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithTimeFunc(t.now),
		jwt.WithIssuer(issuer), jwt.WithAudience(audience), jwt.WithExpirationRequired())
	if err != nil {
		return Claims{}, time.Time{}, err
	}
	id, err := strconv.ParseInt(tc.Subject, 10, 64)
	if err != nil {
		return Claims{}, time.Time{}, fmt.Errorf("invalid subject: %w", err)
	}
	if tc.Role != RoleAdmin && tc.Role != RoleDriver {
		return Claims{}, time.Time{}, errors.New("invalid role")
	}
	return Claims{UserID: id, Role: tc.Role}, tc.ExpiresAt.Time, nil
}
