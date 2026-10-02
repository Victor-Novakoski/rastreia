package auth

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// fakeSessions keeps refresh tokens in memory, with the same rules as the SQL.
type fakeSessions struct {
	mu     sync.Mutex
	tokens []*store.RefreshToken
	roles  map[int64]string
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{roles: map[int64]string{7: RoleDriver}}
}

func (f *fakeSessions) CreateRefreshToken(_ context.Context, arg store.CreateRefreshTokenParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens = append(f.tokens, &store.RefreshToken{
		ID: int64(len(f.tokens) + 1), UserID: arg.UserID, FamilyID: arg.FamilyID,
		TokenHash: arg.TokenHash, ExpiresAt: arg.ExpiresAt,
	})
	return nil
}

func (f *fakeSessions) GetRefreshToken(_ context.Context, hash []byte) (store.GetRefreshTokenRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.tokens {
		if bytes.Equal(t.TokenHash, hash) {
			return store.GetRefreshTokenRow{
				ID: t.ID, UserID: t.UserID, FamilyID: t.FamilyID, ExpiresAt: t.ExpiresAt,
				UsedAt: t.UsedAt, RevokedAt: t.RevokedAt, Role: f.roles[t.UserID], CarrierID: 1,
			}, nil
		}
	}
	return store.GetRefreshTokenRow{}, pgx.ErrNoRows
}

func (f *fakeSessions) UseRefreshToken(_ context.Context, id int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := f.tokens[id-1]
	if t.UsedAt != nil || t.RevokedAt != nil {
		return 0, nil
	}
	now := time.Now()
	t.UsedAt = &now
	return 1, nil
}

func (f *fakeSessions) RevokeRefreshFamily(_ context.Context, family string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	for _, t := range f.tokens {
		if t.FamilyID == family && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func TestSessions_Rotate(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newFakeSessions(), time.Hour)

	first, err := s.Start(ctx, 7)
	require.NoError(t, err)

	claims, second, err := s.Rotate(ctx, first.Value)
	require.NoError(t, err)
	assert.Equal(t, Claims{UserID: 7, Role: RoleDriver, CarrierID: 1}, claims)
	assert.NotEqual(t, first.Value, second.Value)

	_, third, err := s.Rotate(ctx, second.Value)
	require.NoError(t, err, "the new token works once")
	assert.NotEmpty(t, third.Value)
}

func TestSessions_ReuseRevokesTheFamily(t *testing.T) {
	ctx := context.Background()
	s := NewSessions(newFakeSessions(), time.Hour)
	first, err := s.Start(ctx, 7)
	require.NoError(t, err)
	_, second, err := s.Rotate(ctx, first.Value)
	require.NoError(t, err)

	// Someone replays the old token: it fails and takes the session down.
	_, _, err = s.Rotate(ctx, first.Value)
	require.ErrorIs(t, err, ErrInvalidSession)
	_, _, err = s.Rotate(ctx, second.Value)
	assert.ErrorIs(t, err, ErrInvalidSession, "the legitimate token is revoked too")
}

func TestSessions_RejectsInvalidTokens(t *testing.T) {
	ctx := context.Background()
	fake := newFakeSessions()
	s := NewSessions(fake, time.Hour)

	_, _, err := s.Rotate(ctx, "unknown")
	assert.ErrorIs(t, err, ErrInvalidSession)

	expired := NewSessions(fake, time.Hour)
	expired.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	old, err := expired.Start(ctx, 7)
	require.NoError(t, err)
	_, _, err = s.Rotate(ctx, old.Value)
	assert.ErrorIs(t, err, ErrInvalidSession, "expired")

	ended, err := s.Start(ctx, 7)
	require.NoError(t, err)
	require.NoError(t, s.End(ctx, ended.Value))
	_, _, err = s.Rotate(ctx, ended.Value)
	assert.ErrorIs(t, err, ErrInvalidSession, "after logout")

	assert.NoError(t, s.End(ctx, "unknown"), "logout with an unknown token is fine")
}

func TestSessions_StoresOnlyTheHash(t *testing.T) {
	fake := newFakeSessions()
	rt, err := NewSessions(fake, time.Hour).Start(context.Background(), 7)
	require.NoError(t, err)
	assert.NotContains(t, string(fake.tokens[0].TokenHash), rt.Value)
	assert.Equal(t, hashToken(rt.Value), fake.tokens[0].TokenHash)
}
