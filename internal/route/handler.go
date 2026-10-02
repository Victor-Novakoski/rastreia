package route

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Today(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, func(driver auth.Claims) (Route, error) {
		return h.svc.Today(r.Context(), driver)
	})
}

type addInput struct {
	Code string `json:"code"`
}

func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	var in addInput
	if !decode(w, r, &in) {
		return
	}
	h.respond(w, r, func(driver auth.Claims) (Route, error) {
		return h.svc.Add(r.Context(), driver, in.Code)
	})
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	h.respond(w, r, func(driver auth.Claims) (Route, error) {
		return h.svc.Remove(r.Context(), driver, id)
	})
}

type reorderInput struct {
	DeliveryIDs []int64 `json:"delivery_ids"`
}

func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	var in reorderInput
	if !decode(w, r, &in) {
		return
	}
	h.respond(w, r, func(driver auth.Claims) (Route, error) {
		return h.svc.Reorder(r.Context(), driver, in.DeliveryIDs)
	})
}

// optimizeInput is where the driver is; both empty orders from the first stop.
type optimizeInput struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func (h *Handler) Optimize(w http.ResponseWriter, r *http.Request) {
	var in optimizeInput
	if !decode(w, r, &in) {
		return
	}
	start, err := in.start()
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	h.respond(w, r, func(driver auth.Claims) (Route, error) {
		return h.svc.Optimize(r.Context(), driver, start)
	})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := httpx.Decode(w, r, dst); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request, fn func(auth.Claims) (Route, error)) {
	driver, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	route, err := fn(driver)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, route)
}
