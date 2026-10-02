package delivery

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
)

func TestCreateIdempotent(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)
	ctx := context.Background()

	first, replayed, err := svc.CreateIdempotent(ctx, owner, "key-1", validInput())
	require.NoError(t, err)
	assert.False(t, replayed)

	again := validInput()
	again.RecipientName = "Maria Souza" // same request after normalization
	second, replayed, err := svc.CreateIdempotent(ctx, owner, "key-1", again)
	require.NoError(t, err)
	assert.True(t, replayed)
	assert.Equal(t, first.ID, second.ID)
	assert.Len(t, fs.deliveries, 1, "a retry does not create another delivery")
	assert.Len(t, fs.events, 1, "nor another event")

	other, _, err := svc.CreateIdempotent(ctx, owner, "key-2", validInput())
	require.NoError(t, err)
	assert.NotEqual(t, first.ID, other.ID, "a new key creates a new delivery")

	otherUser, replayed, err := svc.CreateIdempotent(ctx, auth.Claims{UserID: 99, Role: auth.RoleCarrier, CarrierID: 1}, "key-1", validInput())
	require.NoError(t, err)
	assert.False(t, replayed, "keys are per user")
	assert.NotEqual(t, first.ID, otherUser.ID)
}

func TestCreateIdempotent_Rejects(t *testing.T) {
	svc := NewService(newFakeStore())
	ctx := context.Background()
	_, _, err := svc.CreateIdempotent(ctx, owner, "key-1", validInput())
	require.NoError(t, err)

	changed := validInput()
	changed.Number = "20"
	cases := map[string]struct {
		key string
		in  CreateInput
	}{
		"key reused with another request": {"key-1", changed},
		"empty key":                       {"", validInput()},
		"key with spaces":                 {"my key", validInput()},
		"key too long":                    {strings.Repeat("k", 256), validInput()},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := svc.CreateIdempotent(ctx, owner, tc.key, tc.in)
			var verr *apperr.ValidationError
			require.ErrorAs(t, err, &verr)
			assert.Contains(t, verr.Fields, "idempotency_key")
		})
	}
}
