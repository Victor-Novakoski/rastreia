package user

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type fakeStore struct {
	users []store.User
}

func (f *fakeStore) CreateUser(_ context.Context, arg store.CreateUserParams) (store.User, error) {
	for _, u := range f.users {
		if u.Email == arg.Email {
			return store.User{}, &pgconn.PgError{Code: "23505"}
		}
	}
	u := store.User{ID: int64(len(f.users) + 1), Name: arg.Name, Email: arg.Email, PasswordHash: arg.PasswordHash, Role: arg.Role}
	f.users = append(f.users, u)
	return u, nil
}

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (store.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return store.User{}, pgx.ErrNoRows
}

func (f *fakeStore) ListUsersByRole(_ context.Context, role string) ([]store.User, error) {
	var out []store.User
	for _, u := range f.users {
		if u.Role == role {
			out = append(out, u)
		}
	}
	return out, nil
}

func TestCreateDriver(t *testing.T) {
	fs := &fakeStore{}
	svc := NewService(fs)

	u, err := svc.CreateDriver(context.Background(), CreateInput{Name: " João ", Email: "Joao@Example.com", Password: "12345678"})
	require.NoError(t, err)
	assert.Equal(t, User{ID: 1, Name: "João", Email: "joao@example.com", Role: auth.RoleDriver}, u)
	assert.True(t, auth.CheckPassword(fs.users[0].PasswordHash, "12345678"), "password is stored hashed")

	_, err = svc.CreateDriver(context.Background(), CreateInput{Name: "Outro", Email: "joao@example.com", Password: "12345678"})
	assert.ErrorIs(t, err, apperr.ErrConflict)

	_, err = svc.CreateDriver(context.Background(), CreateInput{Name: "", Email: "x", Password: "123"})
	var verr *apperr.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Len(t, verr.Fields, 3)

	drivers, err := svc.ListDrivers(context.Background())
	require.NoError(t, err)
	assert.Len(t, drivers, 1)
}

func TestEnsureAdmin_IsIdempotent(t *testing.T) {
	fs := &fakeStore{}
	svc := NewService(fs)
	in := CreateInput{Name: "Admin", Email: "admin@example.com", Password: "admin12345"}

	require.NoError(t, svc.EnsureAdmin(context.Background(), in))
	require.NoError(t, svc.EnsureAdmin(context.Background(), in))
	require.Len(t, fs.users, 1)
	assert.Equal(t, auth.RoleAdmin, fs.users[0].Role)
}
