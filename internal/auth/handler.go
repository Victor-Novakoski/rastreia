package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Victor-Novakoski/rastreia/internal/httpx"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

type UserFinder interface {
	GetUserByEmail(ctx context.Context, email string) (store.User, error)
}

// RefreshCookie is the cookie that carries the refresh token. It is only
// sent to /auth/... and never readable by JavaScript (SECURITY.md #18).
const RefreshCookie = "rastreia_refresh"

// CookieOptions says how the refresh cookie is set and who may use it.
type CookieOptions struct {
	// AllowedOrigins must contain the Origin of /auth/refresh and
	// /auth/logout requests. With SameSite=Strict, it is the CSRF defense
	// (SECURITY.md #10).
	AllowedOrigins []string
}

type Handler struct {
	users    UserFinder
	tokens   *Tokens
	guard    *LoginGuard
	sessions *Sessions
	cookie   CookieOptions
}

func NewHandler(users UserFinder, tokens *Tokens, guard *LoginGuard, sessions *Sessions, cookie CookieOptions) *Handler {
	return &Handler{users: users, tokens: tokens, guard: guard, sessions: sessions, cookie: cookie}
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
	// ExpiresIn is the access token lifetime in seconds.
	ExpiresIn int `json:"expires_in"`
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
	refresh, err := h.sessions.Start(r.Context(), user.ID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	h.respondWithTokens(w, Claims{UserID: user.ID, Role: user.Role}, refresh)
}

// Refresh trades the refresh cookie for a new access token and a new cookie.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if !h.originAllowed(r) {
		httpx.Error(w, http.StatusForbidden, "origin not allowed")
		return
	}
	c, err := r.Cookie(RefreshCookie)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "missing session")
		return
	}
	claims, refresh, err := h.sessions.Rotate(r.Context(), c.Value)
	if errors.Is(err, ErrInvalidSession) {
		h.clearCookie(w)
		httpx.Error(w, http.StatusUnauthorized, "invalid session")
		return
	}
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	h.respondWithTokens(w, claims, refresh)
}

// Logout revokes the session and clears the cookie. It always answers 204,
// even without a cookie, so the front can call it unconditionally.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if !h.originAllowed(r) {
		httpx.Error(w, http.StatusForbidden, "origin not allowed")
		return
	}
	if c, err := r.Cookie(RefreshCookie); err == nil {
		if err := h.sessions.End(r.Context(), c.Value); err != nil {
			httpx.WriteError(w, err)
			return
		}
	}
	h.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) respondWithTokens(w http.ResponseWriter, claims Claims, refresh RefreshToken) {
	token, err := h.tokens.Issue(claims)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookie,
		Value:    refresh.Value,
		Path:     "/auth",
		Expires:  refresh.ExpiresAt,
		HttpOnly: true,
		Secure:   true, // browsers accept it on http://localhost too
		SameSite: http.SameSiteStrictMode,
	})
	httpx.JSON(w, http.StatusOK, loginResponse{
		Token: token, Role: claims.Role, ExpiresIn: int(h.tokens.ttl / time.Second),
	})
}

func (h *Handler) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookie,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true, // browsers accept it on http://localhost too
		SameSite: http.SameSiteStrictMode,
	})
}

// originAllowed rejects requests without an Origin from the allowed list.
// Browsers always send Origin on POST, so a missing one is not a browser
// on our front-end.
func (h *Handler) originAllowed(r *http.Request) bool {
	return slices.Contains(h.cookie.AllowedOrigins, r.Header.Get("Origin"))
}

// emailFingerprint identifies an e-mail in logs without writing it in full.
func emailFingerprint(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:6])
}
