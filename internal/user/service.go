// Package user manages carriers and the people who log in: the people who
// run a carrier and its drivers.
package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type Store interface {
	CreateUser(ctx context.Context, arg store.CreateUserParams) (store.User, error)
	CreateCarrierWithOwner(ctx context.Context, arg store.CreateCarrierWithOwnerParams) (store.User, error)
	GetCarrier(ctx context.Context, id int64) (store.Carrier, error)
	GetUserByEmail(ctx context.Context, email string) (store.User, error)
	GetUserByID(ctx context.Context, id int64) (store.User, error)
	ListCarrierUsersByRole(ctx context.Context, arg store.ListCarrierUsersByRoleParams) ([]store.User, error)
}

// User is the public view of a user: it never carries the password hash.
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CarrierID int64     `json:"carrier_id"`
	CreatedAt time.Time `json:"created_at"`
}

func fromStore(u store.User) User {
	return User{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, CarrierID: u.CarrierID, CreatedAt: u.CreatedAt}
}

type CreateInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignUpInput opens a new carrier with the person who runs it.
type SignUpInput struct {
	CarrierName string  `json:"carrier_name"`
	Document    *string `json:"document"`
	CreateInput
}

// Carrier is a transportadora: a tenant with its own drivers and deliveries.
type Carrier struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Document *string `json:"document"`
}

// Me is the logged-in user with the carrier they belong to.
type Me struct {
	User
	Carrier Carrier `json:"carrier"`
}

const (
	maxName  = 120
	maxEmail = 254

	// DemoCarrier is the carrier created for the account in ADMIN_EMAIL.
	DemoCarrier = "Transportadora Demo"
)

type Service struct {
	store Store
}

func NewService(s Store) *Service {
	return &Service{store: s}
}

// SignUp creates a carrier and the account of the person who runs it.
func (s *Service) SignUp(ctx context.Context, in SignUpInput) (User, error) {
	return s.signUp(ctx, in, true)
}

// signUp creates the carrier. strict is off only for the demo account, whose
// password comes from configuration (see validate).
func (s *Service) signUp(ctx context.Context, in SignUpInput, strict bool) (User, error) {
	in.CarrierName = strings.TrimSpace(in.CarrierName)
	if in.Document != nil {
		doc := normalizeCNPJ(*in.Document)
		in.Document = &doc
		if doc == "" {
			in.Document = nil
		}
	}
	v := apperr.Validator{}
	v.Check(in.CarrierName != "", "carrier_name", "is required")
	v.Check(utf8.RuneCountInString(in.CarrierName) <= maxName, "carrier_name", "must have at most 120 characters")
	v.Check(in.Document == nil || validCNPJ(*in.Document), "document", "must be a valid CNPJ")
	in.CreateInput = s.validate(v, in.CreateInput, strict)
	if err := v.Err(); err != nil {
		return User{}, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return User{}, err
	}
	u, err := s.store.CreateCarrierWithOwner(ctx, store.CreateCarrierWithOwnerParams{
		CarrierName: in.CarrierName, Document: in.Document,
		Name: in.Name, Email: in.Email, PasswordHash: hash,
	})
	if err := conflict(err); err != nil {
		return User{}, err
	}
	return fromStore(u), nil
}

// CreateDriver adds a driver to the carrier.
func (s *Service) CreateDriver(ctx context.Context, carrierID int64, in CreateInput) (User, error) {
	v := apperr.Validator{}
	in = s.validate(v, in, true)
	if err := v.Err(); err != nil {
		return User{}, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return User{}, err
	}
	u, err := s.store.CreateUser(ctx, store.CreateUserParams{
		CarrierID: carrierID, Name: in.Name, Email: in.Email, PasswordHash: hash, Role: auth.RoleDriver,
	})
	if err := conflict(err); err != nil {
		return User{}, err
	}
	return fromStore(u), nil
}

// Me returns the logged-in user and their carrier.
func (s *Service) Me(ctx context.Context, actor auth.Claims) (Me, error) {
	u, err := s.store.GetUserByID(ctx, actor.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Me{}, apperr.ErrNotFound
	}
	if err != nil {
		return Me{}, err
	}
	c, err := s.store.GetCarrier(ctx, u.CarrierID)
	if err != nil {
		return Me{}, err
	}
	return Me{User: fromStore(u), Carrier: Carrier{ID: c.ID, Name: c.Name, Document: c.Document}}, nil
}

// ListDrivers lists the carrier's drivers; other carriers' never leave the database.
func (s *Service) ListDrivers(ctx context.Context, carrierID int64) ([]User, error) {
	rows, err := s.store.ListCarrierUsersByRole(ctx, store.ListCarrierUsersByRoleParams{
		CarrierID: carrierID, Role: auth.RoleDriver,
	})
	if err != nil {
		return nil, err
	}
	users := make([]User, len(rows))
	for i, u := range rows {
		users[i] = fromStore(u)
	}
	return users, nil
}

// EnsureDemoCarrier creates a demo carrier run by the given account when no
// user has that e-mail yet, so a fresh database can be logged into.
func (s *Service) EnsureDemoCarrier(ctx context.Context, in CreateInput) error {
	_, err := s.store.GetUserByEmail(ctx, normalizeEmail(in.Email))
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	// The password comes from configuration, which already refuses the
	// development default in production, so only the length rule applies.
	_, err = s.signUp(ctx, SignUpInput{CarrierName: DemoCarrier, CreateInput: in}, false)
	if errors.Is(err, apperr.ErrConflict) {
		return nil // another instance, starting at the same time, created it
	}
	return err
}

// validate normalizes and checks the fields shared by every account. strict
// also refuses common passwords and passwords made from the e-mail; the demo
// account skips it, since its development password is both.
func (s *Service) validate(v apperr.Validator, in CreateInput, strict bool) CreateInput {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = normalizeEmail(in.Email)
	v.Check(in.Name != "", "name", "is required")
	v.Check(utf8.RuneCountInString(in.Name) <= maxName, "name", "must have at most 120 characters")
	v.Check(len(in.Email) <= maxEmail && validEmail(in.Email), "email", "must be a valid e-mail")
	problem := auth.PasswordProblem(in.Password)
	switch {
	case !strict && problem == "is too common":
		problem = ""
	case strict && problem == "" && auth.PasswordHasEmail(in.Password, in.Email):
		problem = "must not contain the e-mail"
	}
	v.Check(problem == "", "password", problem)
	return in
}

// conflict turns unique violations into a conflict naming what is taken.
func conflict(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	if pgErr.ConstraintName == "carriers_document_idx" {
		return fmt.Errorf("%w: CNPJ already in use", apperr.ErrConflict)
	}
	return fmt.Errorf("%w: e-mail already in use", apperr.ErrConflict)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

// normalizeCNPJ drops the punctuation of a CNPJ and uppercases it: since
// July 2026 the first 12 characters may be letters.
func normalizeCNPJ(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(" .-/", r) {
			return -1
		}
		return unicode.ToUpper(r)
	}, s)
}

// validCNPJ checks the format (12 letters or digits, then 2 digits) and both
// check digits. Each character counts as its ASCII code minus 48, which
// keeps the old all-digits rule.
func validCNPJ(doc string) bool {
	if len(doc) != 14 || strings.Count(doc, doc[:1]) == 14 {
		return false
	}
	for i, c := range []byte(doc) {
		digit := c >= '0' && c <= '9'
		if !digit && (i >= 12 || c < 'A' || c > 'Z') {
			return false
		}
	}
	check := func(n int) byte {
		weights := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}[13-n:]
		sum := 0
		for i, w := range weights {
			sum += int(doc[i]-'0') * w
		}
		if r := sum % 11; r >= 2 {
			return byte('0' + 11 - r)
		}
		return '0'
	}
	return check(12) == doc[12] && check(13) == doc[13]
}
