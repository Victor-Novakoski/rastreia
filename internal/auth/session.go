package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// ErrInvalidSession covers every refresh token that cannot be used: unknown,
// expired, revoked or already used. The client just has to log in again.
var ErrInvalidSession = errors.New("invalid session")

type SessionStore interface {
	CreateRefreshToken(ctx context.Context, arg store.CreateRefreshTokenParams) error
	GetRefreshToken(ctx context.Context, tokenHash []byte) (store.GetRefreshTokenRow, error)
	UseRefreshToken(ctx context.Context, id int64) (int64, error)
	RevokeRefreshFamily(ctx context.Context, familyID string) error
}

// Sessions issues and rotates refresh tokens (SECURITY.md #14). The token is
// 32 random bytes; only its SHA-256 goes to the database.
type Sessions struct {
	store SessionStore
	ttl   time.Duration
	now   func() time.Time
}

func NewSessions(s SessionStore, ttl time.Duration) *Sessions {
	return &Sessions{store: s, ttl: ttl, now: time.Now}
}

// RefreshToken is a new token to hand to the client.
type RefreshToken struct {
	Value     string
	ExpiresAt time.Time
}

// Start opens a new session (a new token family) after a login.
func (s *Sessions) Start(ctx context.Context, userID int64) (RefreshToken, error) {
	family, err := randomString(16)
	if err != nil {
		return RefreshToken{}, err
	}
	return s.issue(ctx, userID, family)
}

// Rotate trades a refresh token for the user's current claims and the next
// token of the same family. A token that was already used means it leaked
// (or the client misbehaves): the whole family is revoked.
func (s *Sessions) Rotate(ctx context.Context, raw string) (Claims, RefreshToken, error) {
	rt, err := s.store.GetRefreshToken(ctx, hashToken(raw))
	if errors.Is(err, pgx.ErrNoRows) {
		return Claims{}, RefreshToken{}, ErrInvalidSession
	}
	if err != nil {
		return Claims{}, RefreshToken{}, fmt.Errorf("get refresh token: %w", err)
	}
	if rt.RevokedAt != nil || !s.now().Before(rt.ExpiresAt) {
		return Claims{}, RefreshToken{}, ErrInvalidSession
	}
	if rt.UsedAt != nil {
		return Claims{}, RefreshToken{}, s.revokeOnReuse(ctx, rt)
	}
	n, err := s.store.UseRefreshToken(ctx, rt.ID)
	if err != nil {
		return Claims{}, RefreshToken{}, fmt.Errorf("use refresh token: %w", err)
	}
	if n == 0 { // another request used it between the read and the update
		return Claims{}, RefreshToken{}, s.revokeOnReuse(ctx, rt)
	}
	next, err := s.issue(ctx, rt.UserID, rt.FamilyID)
	if err != nil {
		return Claims{}, RefreshToken{}, err
	}
	return Claims{UserID: rt.UserID, Role: rt.Role}, next, nil
}

// End revokes the session of the given token (logout). Unknown tokens are ignored.
func (s *Sessions) End(ctx context.Context, raw string) error {
	rt, err := s.store.GetRefreshToken(ctx, hashToken(raw))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get refresh token: %w", err)
	}
	return s.store.RevokeRefreshFamily(ctx, rt.FamilyID)
}

func (s *Sessions) revokeOnReuse(ctx context.Context, rt store.GetRefreshTokenRow) error {
	slog.Warn("refresh token reused, session revoked", "user_id", rt.UserID)
	if err := s.store.RevokeRefreshFamily(ctx, rt.FamilyID); err != nil {
		return fmt.Errorf("revoke refresh family: %w", err)
	}
	return ErrInvalidSession
}

func (s *Sessions) issue(ctx context.Context, userID int64, family string) (RefreshToken, error) {
	value, err := randomString(32)
	if err != nil {
		return RefreshToken{}, err
	}
	expires := s.now().Add(s.ttl)
	err = s.store.CreateRefreshToken(ctx, store.CreateRefreshTokenParams{
		UserID: userID, FamilyID: family, TokenHash: hashToken(value), ExpiresAt: expires,
	})
	if err != nil {
		return RefreshToken{}, fmt.Errorf("create refresh token: %w", err)
	}
	return RefreshToken{Value: value, ExpiresAt: expires}, nil
}

func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
