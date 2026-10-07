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
	// /auth/logout requests, and of /auth/login and /auth/signup when a
	// browser sends them. With SameSite=Strict, it is the CSRF defense
	// (SECURITY.md #10).
	AllowedOrigins []string
}

type Handler struct {
	users    UserFinder
	tokens   *Tokens
	guard    Guard
	sessions *Sessions
	cookie   CookieOptions
}

func NewHandler(users UserFinder, tokens *Tokens, guard Guard, sessions *Sessions, cookie CookieOptions) *Handler {
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
	wait, err := h.guard.Check(r.Context(), email)
	if err != nil {
		// Without the count there is no brute-force protection, so the
		// login waits for it instead of going ahead.
		slog.Error("login guard", "err", err)
		httpx.Error(w, http.StatusServiceUnavailable, "login unavailable, try again later")
		return
	}
	if wait > 0 {
		httpx.AddLogAttrs(r.Context(), slog.String("reason", "login locked"), slog.String("email_hash", emailFingerprint(email)))
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
		if err := h.guard.Fail(r.Context(), email); err != nil {
			slog.Error("login guard", "err", err)
		}
		httpx.AddLogAttrs(r.Context(), slog.String("reason", "wrong e-mail or password"), slog.String("email_hash", emailFingerprint(email)))
		httpx.Error(w, http.StatusUnauthorized, "invalid e-mail or password")
		return
	}
	if err := h.guard.Success(r.Context(), email); err != nil {
		slog.Error("login guard", "err", err)
	}
	h.StartSession(w, r, Claims{UserID: user.ID, Role: user.Role, CarrierID: user.CarrierID})
}

// StartSession opens a session for a user who just proved who they are (a
// login or a sign-up) and answers with the access token and refresh cookie.
func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request, c Claims) {
	refresh, err := h.sessions.Start(r.Context(), c.UserID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	h.respondWithTokens(w, c, refresh)
}

// RefuseForeignOrigin answers 403, before anything is done, to a browser
// sending the request from a site not in AllowedOrigins. /auth/login and
// /auth/signup set the refresh cookie: without this, another site could post
// a form that logs the victim's browser into the attacker's account (login
// CSRF). Clients that are not browsers send no Origin and go through.
func (h *Handler) RefuseForeignOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.originRefused(w, r, false) {
			next.ServeHTTP(w, r)
		}
	})
}

// Refresh trades the refresh cookie for a new access token and a new cookie.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if h.originRefused(w, r, true) {
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
	if h.originRefused(w, r, true) {
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

// originRefused answers 403 when the request's Origin is not in the allowed
// list. With required, a request without Origin is refused too: browsers
// always send it on POST, so it is not a browser on our front-end, and only
// a browser has the cookie.
func (h *Handler) originRefused(w http.ResponseWriter, r *http.Request, required bool) bool {
	origin := r.Header.Get("Origin")
	if slices.Contains(h.cookie.AllowedOrigins, origin) || (origin == "" && !required) {
		return false
	}
	httpx.AddLogAttrs(r.Context(), slog.String("reason", "origin not allowed"))
	httpx.Error(w, http.StatusForbidden, "origin not allowed")
	return true
}

// emailFingerprint identifies an e-mail in logs without writing it in full.
func emailFingerprint(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:6])
}
