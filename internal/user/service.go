// Package user manages the people who log in: admins and drivers.
package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type Store interface {
	CreateUser(ctx context.Context, arg store.CreateUserParams) (store.User, error)
	GetUserByEmail(ctx context.Context, email string) (store.User, error)
	ListUsersByRole(ctx context.Context, role string) ([]store.User, error)
}

// User is the public view of a user: it never carries the password hash.
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func fromStore(u store.User) User {
	return User{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, CreatedAt: u.CreatedAt}
}

type CreateInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

const (
	maxName  = 120
	maxEmail = 254
)

type Service struct {
	store Store
}

func NewService(s Store) *Service {
	return &Service{store: s}
}

func (s *Service) CreateDriver(ctx context.Context, in CreateInput) (User, error) {
	return s.create(ctx, in, auth.RoleDriver, true)
}

func (s *Service) ListDrivers(ctx context.Context) ([]User, error) {
	rows, err := s.store.ListUsersByRole(ctx, auth.RoleDriver)
	if err != nil {
		return nil, err
	}
	users := make([]User, len(rows))
	for i, u := range rows {
		users[i] = fromStore(u)
	}
	return users, nil
}

// EnsureAdmin creates the first admin when no user has that e-mail yet, so a
// fresh database can be logged into.
func (s *Service) EnsureAdmin(ctx context.Context, in CreateInput) error {
	_, err := s.store.GetUserByEmail(ctx, normalizeEmail(in.Email))
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	// The admin password comes from configuration, which already refuses the
	// development default in production, so only the length rule applies.
	_, err = s.create(ctx, in, auth.RoleAdmin, false)
	return err
}

func (s *Service) create(ctx context.Context, in CreateInput, role string, rejectCommon bool) (User, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = normalizeEmail(in.Email)

	v := apperr.Validator{}
	v.Check(in.Name != "", "name", "is required")
	v.Check(len(in.Name) <= maxName, "name", "must have at most 120 characters")
	v.Check(len(in.Email) <= maxEmail && validEmail(in.Email), "email", "must be a valid e-mail")
	problem := auth.PasswordProblem(in.Password)
	if !rejectCommon && problem == "is too common" {
		problem = ""
	}
	v.Check(problem == "", "password", problem)
	if err := v.Err(); err != nil {
		return User{}, err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return User{}, err
	}
	u, err := s.store.CreateUser(ctx, store.CreateUserParams{
		Name: in.Name, Email: in.Email, PasswordHash: hash, Role: role,
	})
	if isUniqueViolation(err) {
		return User{}, fmt.Errorf("%w: e-mail already in use", apperr.ErrConflict)
	}
	if err != nil {
		return User{}, err
	}
	return fromStore(u), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
