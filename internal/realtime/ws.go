package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/Victor-Novakoski/rastreia/internal/httpx"
)

// Close codes the front reacts to. 4000-4999 are free for applications.
const (
	// StatusUnauthorized asks the browser to renew its token and reconnect.
	StatusUnauthorized websocket.StatusCode = 4001
)

const (
	authTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
	// Proxies and load balancers drop connections that stay silent for too
	// long; a ping every 30s keeps them open and finds dead ones.
	pingInterval = 30 * time.Second
	// Browsers only send the auth message, so anything bigger is abuse.
	readLimit = 4 << 10
)

type Options struct {
	// Origins are the front-end origins allowed to connect, like the CORS list.
	Origins []string
	// MaxPerIP caps open connections per client IP. Zero uses 20.
	MaxPerIP int
}

// Server streams broker topics to WebSocket clients.
type Server struct {
	broker   Broker
	shutdown context.Context
	origins  []string
	maxPerIP int

	mu    sync.Mutex
	perIP map[string]int
}

// NewServer creates a Server. Its connections are closed when shutdown ends,
// since http.Server.Shutdown does not wait for or close them.
func NewServer(shutdown context.Context, b Broker, opts Options) *Server {
	if opts.MaxPerIP == 0 {
		opts.MaxPerIP = 20
	}
	return &Server{
		broker: b, shutdown: shutdown, origins: opts.Origins,
		maxPerIP: opts.MaxPerIP, perIP: map[string]int{},
	}
}

// Authorize checks the token a browser sends as its first message and says
// when the connection must close because the token expired.
type Authorize func(token string) (expires time.Time, err error)

type authMessage struct {
	Token string `json:"token"`
}

// Stream upgrades the request and sends every message published on topic
// until the client leaves. With authorize set, the first message from the
// client must be {"token": "..."}; browsers cannot send an Authorization
// header on a WebSocket, and a token in the URL would end up in logs.
func (s *Server) Stream(w http.ResponseWriter, r *http.Request, topic string, authorize Authorize) {
	ip := clientIP(r)
	if !s.acquire(ip) {
		httpx.Error(w, http.StatusTooManyRequests, "too many open connections")
		return
	}
	defer s.release(ip)

	// The server's read and write timeouts are for plain requests; a
	// WebSocket stays open and has its own ping and write timeouts.
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Time{})
	_ = rc.SetWriteDeadline(time.Time{})

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.origins})
	if err != nil {
		return // Accept already answered the request
	}
	defer func() { _ = c.CloseNow() }()
	c.SetReadLimit(readLimit)

	ctx := r.Context()
	var expires <-chan time.Time
	if authorize != nil {
		exp, err := readAuth(ctx, c, authorize)
		if err != nil {
			_ = c.Close(StatusUnauthorized, "unauthorized")
			return
		}
		t := time.NewTimer(time.Until(exp))
		defer t.Stop()
		expires = t.C
	}

	msgs, err := s.broker.Subscribe(ctx, topic)
	if err != nil {
		slog.Error("realtime subscribe", "topic", topic, "err", err)
		_ = c.Close(websocket.StatusInternalError, "")
		return
	}
	// From here on the client has nothing to say; CloseRead handles pings and
	// the close handshake, and ends ctx if the client sends anything else.
	ctx = c.CloseRead(ctx)

	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				_ = c.Close(websocket.StatusTryAgainLater, "too slow")
				return
			}
			if err := write(ctx, c, msg); err != nil {
				return
			}
		case <-ping.C:
			pctx, pcancel := context.WithTimeout(ctx, writeTimeout)
			err := c.Ping(pctx)
			pcancel()
			if err != nil {
				return
			}
		case <-expires:
			_ = c.Close(StatusUnauthorized, "token expired")
			return
		case <-s.shutdown.Done():
			_ = c.Close(websocket.StatusGoingAway, "server shutting down")
			return
		case <-ctx.Done():
			return
		}
	}
}

func readAuth(ctx context.Context, c *websocket.Conn, authorize Authorize) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, authTimeout)
	defer cancel()
	typ, data, err := c.Read(ctx)
	if err != nil {
		return time.Time{}, err
	}
	if typ != websocket.MessageText {
		return time.Time{}, errors.New("auth message must be text")
	}
	var m authMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return time.Time{}, err
	}
	return authorize(m.Token)
}

func write(ctx context.Context, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageText, msg)
}

func (s *Server) acquire(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.perIP[ip] >= s.maxPerIP {
		return false
	}
	s.perIP[ip]++
	return true
}

func (s *Server) release(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.perIP[ip]--; s.perIP[ip] <= 0 {
		delete(s.perIP, ip)
	}
}

// clientIP is r.RemoteAddr without the port. Behind a proxy the server's
// trustedProxy middleware has already put the real client IP there.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
