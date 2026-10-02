package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/httpx"
	"github.com/Victor-Novakoski/rastreia/internal/realtime"
)

// Publisher sends a message to everyone watching a topic live.
type Publisher interface {
	Publish(ctx context.Context, topic string, msg []byte) error
}

// TopicPanel carries a PanelChange for every delivery created or changed.
const TopicPanel = "deliveries"

// TrackingTopic carries the public Tracking of one delivery after each change.
func TrackingTopic(code string) string {
	return "tracking:" + code
}

// PanelChange tells the admin panel which delivery to reload. It carries no
// personal data, so a leak of the stream says little.
type PanelChange struct {
	DeliveryID int64  `json:"delivery_id"`
	Status     string `json:"status"`
}

// WithPublisher makes the service announce changes live. Without it nothing
// is published.
func (s *Service) WithPublisher(p Publisher) *Service {
	s.pub = p
	return s
}

// announce publishes a change after it is committed. A failure is only
// logged: the change is saved, and browsers catch up on their next load.
func (s *Service) announce(ctx context.Context, id int64, code, status string, public bool) {
	if s.pub == nil {
		return
	}
	s.publish(ctx, TopicPanel, PanelChange{DeliveryID: id, Status: status})
	if !public {
		return
	}
	t, err := s.Track(ctx, code)
	if err != nil {
		slog.Error("realtime tracking", "delivery_id", id, "err", err)
		return
	}
	s.publish(ctx, TrackingTopic(code), t)
}

func (s *Service) publish(ctx context.Context, topic string, v any) {
	msg, err := json.Marshal(v)
	if err == nil {
		err = s.pub.Publish(ctx, topic, msg)
	}
	if err != nil {
		slog.Error("realtime publish", "topic", topic, "err", err)
	}
}

// LiveHandler serves the WebSocket routes.
type LiveHandler struct {
	svc    *Service
	live   *realtime.Server
	tokens *auth.Tokens
}

func NewLiveHandler(svc *Service, live *realtime.Server, tokens *auth.Tokens) *LiveHandler {
	return &LiveHandler{svc: svc, live: live, tokens: tokens}
}

// Track streams the public tracking of one delivery: after each change it
// sends the same body as GET /public/tracking/{code}. Unknown and expired
// codes get the same 404 as there.
func (h *LiveHandler) Track(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Track(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	h.live.Stream(w, r, TrackingTopic(t.TrackingCode), nil)
}

// Panel streams a PanelChange for every delivery to admins. The first
// message from the browser must carry an admin access token.
func (h *LiveHandler) Panel(w http.ResponseWriter, r *http.Request) {
	h.live.Stream(w, r, TopicPanel, func(token string) (time.Time, error) {
		c, exp, err := h.tokens.ParseExpiry(token)
		if err != nil {
			return time.Time{}, err
		}
		if c.Role != auth.RoleAdmin {
			return time.Time{}, errors.New("admins only")
		}
		return exp, nil
	})
}
