package user

import (
	"net/http"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/httpx"
)

// SessionStarter logs a user in, answering with tokens like /auth/login.
type SessionStarter interface {
	StartSession(w http.ResponseWriter, r *http.Request, c auth.Claims)
}

type Handler struct {
	svc      *Service
	sessions SessionStarter
}

func NewHandler(svc *Service, sessions SessionStarter) *Handler {
	return &Handler{svc: svc, sessions: sessions}
}

// SignUp opens a carrier and logs its owner in, with the same answer as
// /auth/login.
func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	var in SignUpInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.svc.SignUp(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	h.sessions.StartSession(w, r, auth.Claims{UserID: u.ID, Role: u.Role, CarrierID: u.CarrierID})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	me, err := h.svc.Me(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, me)
}

func (h *Handler) CreateDriver(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	var in CreateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.svc.CreateDriver(r.Context(), actor.CarrierID, in)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, u)
}

func (h *Handler) ListDrivers(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	users, err := h.svc.ListDrivers(r.Context(), actor.CarrierID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, users)
}

// actorFrom returns the caller set by auth.Authenticate.
func actorFrom(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
	}
	return c, ok
}
