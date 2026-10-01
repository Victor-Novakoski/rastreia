package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Victor-Novakoski/rastreia/internal/httpx"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type UserFinder interface {
	GetUserByEmail(ctx context.Context, email string) (store.User, error)
}

type Handler struct {
	users  UserFinder
	tokens *Tokens
	guard  *LoginGuard
}

func NewHandler(users UserFinder, tokens *Tokens, guard *LoginGuard) *Handler {
	return &Handler{users: users, tokens: tokens, guard: guard}
}

// dummyHash is compared against when the e-mail does not exist, so a login
// for an unknown e-mail takes as long as one with a wrong password.
var dummyHash, _ = HashPassword("dummy-password-for-timing")

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if wait := h.guard.Check(email); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		httpx.Error(w, http.StatusTooManyRequests, "too many failed attempts, try again later")
		return
	}

	user, err := h.users.GetUserByEmail(r.Context(), email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, err)
		return
	}
	hash := user.PasswordHash
	if err != nil {
		hash = dummyHash
	}
	// Same answer for unknown e-mail and wrong password, so the endpoint does
	// not reveal which e-mails exist.
	if !CheckPassword(hash, req.Password) || err != nil {
		h.guard.Fail(email)
		slog.Warn("login failed", "email_hash", emailFingerprint(email), "ip", r.RemoteAddr)
		httpx.Error(w, http.StatusUnauthorized, "invalid e-mail or password")
		return
	}
	h.guard.Success(email)
	token, err := h.tokens.Issue(Claims{UserID: user.ID, Role: user.Role})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, loginResponse{Token: token, Role: user.Role})
}

// emailFingerprint identifies an e-mail in logs without writing it in full.
func emailFingerprint(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:6])
}
