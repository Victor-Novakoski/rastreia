package delivery

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

// Create accepts an optional Idempotency-Key header; a retry with the same
// key answers with the delivery created the first time and sets
// Idempotent-Replayed: true.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	var in CreateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var (
		d        Delivery
		replayed bool
		err      error
	)
	if key, sent := r.Header["Idempotency-Key"]; sent {
		d, replayed, err = h.svc.CreateIdempotent(r.Context(), actor, key[0], in)
	} else {
		d, err = h.svc.Create(r.Context(), actor, in)
	}
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if replayed {
		w.Header().Set("Idempotent-Replayed", "true")
	}
	httpx.JSON(w, http.StatusCreated, d)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	list, err := h.svc.List(r.Context(), actor.CarrierID, listInput(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

// ListMine lists the deliveries assigned to the driver making the request.
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	list, err := h.svc.ListForDriver(r.Context(), actor.UserID, listInput(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) AddEvent(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in EventInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ev, err := h.svc.AddEvent(r.Context(), actor, id, in)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, ev)
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	events, err := h.svc.ListEvents(r.Context(), actor, id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, events)
}

// Track is the public tracking page; it needs no login.
func (h *Handler) Track(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Track(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func listInput(r *http.Request) ListInput {
	q := r.URL.Query()
	in := ListInput{}
	if st := q.Get("status"); st != "" {
		in.Status = &st
	}
	in.Page, _ = strconv.Atoi(q.Get("page"))
	in.Size, _ = strconv.Atoi(q.Get("size"))
	return in
}

// actorFrom returns the caller set by auth.Authenticate.
func actorFrom(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "not authenticated")
	}
	return c, ok
}

// Summary feeds the carrier's dashboard.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	sum, err := h.svc.Summary(r.Context(), actor.CarrierID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	d, err := h.svc.Get(r.Context(), actor, id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in UpdateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	d, err := h.svc.Update(r.Context(), actor, id, in)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}
