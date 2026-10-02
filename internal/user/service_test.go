package user

import (
	"context"
	"strings"
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
	users    []store.User
	carriers []store.Carrier
}

func (f *fakeStore) CreateUser(_ context.Context, arg store.CreateUserParams) (store.User, error) {
	for _, u := range f.users {
		if u.Email == arg.Email {
			return store.User{}, &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"}
		}
	}
	u := store.User{
		ID: int64(len(f.users) + 1), CarrierID: arg.CarrierID, Name: arg.Name, Email: arg.Email,
		PasswordHash: arg.PasswordHash, Role: arg.Role,
	}
	f.users = append(f.users, u)
	return u, nil
}

func (f *fakeStore) CreateCarrierWithOwner(ctx context.Context, arg store.CreateCarrierWithOwnerParams) (store.User, error) {
	for _, c := range f.carriers {
		if c.Document != nil && arg.Document != nil && *c.Document == *arg.Document {
			return store.User{}, &pgconn.PgError{Code: "23505", ConstraintName: "carriers_document_idx"}
		}
	}
	for _, u := range f.users {
		if u.Email == arg.Email {
			return store.User{}, &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"}
		}
	}
	c := store.Carrier{ID: int64(len(f.carriers) + 1), Name: arg.CarrierName, Document: arg.Document}
	f.carriers = append(f.carriers, c)
	return f.CreateUser(ctx, store.CreateUserParams{
		CarrierID: c.ID, Name: arg.Name, Email: arg.Email, PasswordHash: arg.PasswordHash, Role: auth.RoleCarrier,
	})
}

func (f *fakeStore) GetCarrier(_ context.Context, id int64) (store.Carrier, error) {
	for _, c := range f.carriers {
		if c.ID == id {
			return c, nil
		}
	}
	return store.Carrier{}, pgx.ErrNoRows
}

func (f *fakeStore) GetUserByID(_ context.Context, id int64) (store.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return store.User{}, pgx.ErrNoRows
}

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (store.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return store.User{}, pgx.ErrNoRows
}

func (f *fakeStore) ListCarrierUsersByRole(_ context.Context, arg store.ListCarrierUsersByRoleParams) ([]store.User, error) {
	var out []store.User
	for _, u := range f.users {
		if u.CarrierID == arg.CarrierID && u.Role == arg.Role {
			out = append(out, u)
		}
	}
	return out, nil
}

func TestCreateDriver(t *testing.T) {
	fs := &fakeStore{}
	svc := NewService(fs)

	u, err := svc.CreateDriver(context.Background(), 1, CreateInput{Name: " João ", Email: "Joao@Example.com", Password: "motorista-forte"})
	require.NoError(t, err)
	assert.Equal(t, User{ID: 1, Name: "João", Email: "joao@example.com", Role: auth.RoleDriver, CarrierID: 1}, u)
	assert.True(t, auth.CheckPassword(fs.users[0].PasswordHash, "motorista-forte"), "password is stored hashed")

	_, err = svc.CreateDriver(context.Background(), 1, CreateInput{Name: "Outro", Email: "joao@example.com", Password: "motorista-forte"})
	assert.ErrorIs(t, err, apperr.ErrConflict)

	_, err = svc.CreateDriver(context.Background(), 1, CreateInput{Name: "", Email: "x", Password: "123"})
	var verr *apperr.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Len(t, verr.Fields, 3)

	drivers, err := svc.ListDrivers(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, drivers, 1)
	drivers, err = svc.ListDrivers(context.Background(), 2)
	require.NoError(t, err)
	assert.Empty(t, drivers, "drivers are listed per carrier")
}

func TestSignUp(t *testing.T) {
	fs := &fakeStore{}
	svc := NewService(fs)
	in := SignUpInput{
		CarrierName: " Expresso Sul ", Document: ptr("11.222.333/0001-81"),
		CreateInput: CreateInput{Name: "Carla", Email: "Carla@Example.com", Password: "transporte-forte"},
	}

	u, err := svc.SignUp(context.Background(), in)
	require.NoError(t, err)
	assert.Equal(t, auth.RoleCarrier, u.Role)
	assert.Equal(t, "carla@example.com", u.Email)
	require.Len(t, fs.carriers, 1)
	assert.Equal(t, "Expresso Sul", fs.carriers[0].Name)
	assert.Equal(t, ptr("11222333000181"), fs.carriers[0].Document, "the CNPJ is stored without punctuation")

	me, err := svc.Me(context.Background(), auth.Claims{UserID: u.ID, Role: u.Role, CarrierID: u.CarrierID})
	require.NoError(t, err)
	assert.Equal(t, "Expresso Sul", me.Carrier.Name)

	again := in
	again.Email = "outra@example.com"
	_, err = svc.SignUp(context.Background(), again)
	require.ErrorIs(t, err, apperr.ErrConflict)
	assert.Contains(t, err.Error(), "CNPJ")

	again.Document = nil
	again.Email = "carla@example.com"
	_, err = svc.SignUp(context.Background(), again)
	require.ErrorIs(t, err, apperr.ErrConflict)
	assert.Contains(t, err.Error(), "e-mail")

	again.Email = "nova@example.com"
	again.Document = ptr("  ")
	_, err = svc.SignUp(context.Background(), again)
	require.NoError(t, err, "a blank CNPJ is the same as none")
}

func TestSignUp_Validation(t *testing.T) {
	svc := NewService(&fakeStore{})
	_, err := svc.SignUp(context.Background(), SignUpInput{
		CarrierName: "", Document: ptr("11.222.333/0001-82"),
		CreateInput: CreateInput{Name: "Carla", Email: "carla@example.com", Password: "transporte-forte"},
	})
	var verr *apperr.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, verr.Fields, "carrier_name")
	assert.Contains(t, verr.Fields, "document")
}

func TestValidCNPJ(t *testing.T) {
	for _, doc := range []string{"11222333000181", "12ABC34501DE35"} {
		assert.True(t, validCNPJ(normalizeCNPJ(doc)), doc)
	}
	assert.True(t, validCNPJ(normalizeCNPJ("12.abc.345/01de-35")), "letters are uppercased")
	for _, doc := range []string{"11222333000182", "1122233300018", "00000000000000", "11222333000A81", "1122233300018A"} {
		assert.False(t, validCNPJ(doc), doc)
	}
}

func TestEnsureDemoCarrier_IsIdempotent(t *testing.T) {
	fs := &fakeStore{}
	svc := NewService(fs)
	in := CreateInput{Name: "Admin", Email: "admin@example.com", Password: "admin12345"}

	require.NoError(t, svc.EnsureDemoCarrier(context.Background(), in))
	require.NoError(t, svc.EnsureDemoCarrier(context.Background(), in))
	require.Len(t, fs.users, 1)
	assert.Equal(t, auth.RoleCarrier, fs.users[0].Role)
	require.Len(t, fs.carriers, 1)
	assert.Equal(t, DemoCarrier, fs.carriers[0].Name)
}

func TestCreateDriver_PasswordPolicy(t *testing.T) {
	svc := NewService(&fakeStore{})
	for _, pw := range []string{"curta123", "1234567890", "Senha12345", strings.Repeat("a", 73)} {
		_, err := svc.CreateDriver(context.Background(), 1, CreateInput{Name: "Ana", Email: "ana@example.com", Password: pw})
		var verr *apperr.ValidationError
		require.ErrorAs(t, err, &verr, pw)
		assert.Contains(t, verr.Fields, "password")
	}
}

func ptr[T any](v T) *T { return &v }
