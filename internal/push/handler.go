package push

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Victor-Novakoski/rastreia/internal/httpx"
)

type Handler struct {
	svc       *Service
	publicKey string
}

// NewHandler serves the push routes; publicKey is the VAPID public key the
// browser needs to subscribe.
func NewHandler(svc *Service, publicKey string) *Handler {
	return &Handler{svc: svc, publicKey: publicKey}
}

func (h *Handler) Key(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"public_key": h.publicKey})
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var in Subscription
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.Subscribe(r.Context(), chi.URLParam(r, "code"), in); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Endpoint string `json:"endpoint"`
	}
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.Unsubscribe(r.Context(), chi.URLParam(r, "code"), in.Endpoint); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
