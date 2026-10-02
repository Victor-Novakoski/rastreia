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
	// RoleCarrier runs a carrier (transportadora): its deliveries and drivers.
	RoleCarrier = "carrier"
	RoleDriver  = "driver"

	issuer   = "rastreia"
	audience = "rastreia-api"
)

// Claims is what the API knows about the caller after authentication.
type Claims struct {
	UserID int64
	Role   string
	// CarrierID is the carrier the user belongs to; every query is scoped by it.
	CarrierID int64
}

type tokenClaims struct {
	Role      string `json:"role"`
	CarrierID int64  `json:"cid"`
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
		Role:      c.Role,
		CarrierID: c.CarrierID,
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
	if tc.Role != RoleCarrier && tc.Role != RoleDriver {
		return Claims{}, time.Time{}, errors.New("invalid role")
	}
	if tc.CarrierID <= 0 {
		return Claims{}, time.Time{}, errors.New("invalid carrier")
	}
	return Claims{UserID: id, Role: tc.Role, CarrierID: tc.CarrierID}, tc.ExpiresAt.Time, nil
}
