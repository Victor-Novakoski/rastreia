package auth

import (
	"context"
	"errors"
	"net/http"
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
}

func NewHandler(users UserFinder, tokens *Tokens) *Handler {
	return &Handler{users: users, tokens: tokens}
}

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
	user, err := h.users.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, err)
		return
	}
	// Same answer for unknown e-mail and wrong password, so the endpoint does
	// not reveal which e-mails exist.
	if err != nil || !CheckPassword(user.PasswordHash, req.Password) {
		httpx.Error(w, http.StatusUnauthorized, "invalid e-mail or password")
		return
	}
	token, err := h.tokens.Issue(Claims{UserID: user.ID, Role: user.Role})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, loginResponse{Token: token, Role: user.Role})
}
