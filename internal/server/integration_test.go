package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/realtime"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/testdb"
	"github.com/Victor-Novakoski/rastreia/internal/user"
)

const testOrigin = "http://localhost:5173"

// testAPI is the real router on a real database, with one token per user.
type testAPI struct {
	t       *testing.T
	h       http.Handler
	q       *store.Queries
	admin   string
	driverA string
	driverB string
	broker  *countingBroker
}

// countingBroker tells tests when a WebSocket is listening, so a change is
// not published before anyone subscribed.
type countingBroker struct {
	*realtime.Local
	subs atomic.Int32
}

func (b *countingBroker) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	defer b.subs.Add(1)
	return b.Local.Subscribe(ctx, topic)
}

func newAPI(t *testing.T, opts Options) *testAPI {
	pool := testdb.New(t)
	q := store.New(pool)
	tokens := auth.NewTokens("test-secret-with-at-least-32-characters", time.Hour)
	broker := &countingBroker{Local: realtime.NewLocal()}
	deliveries := delivery.NewService(delivery.NewPGStore(pool)).WithPublisher(broker)
	live := realtime.NewServer(t.Context(), broker, realtime.Options{Origins: []string{testOrigin}})
	h := New(Deps{
		Tokens:     tokens,
		Auth:       auth.NewHandler(q, tokens, auth.NewLoginGuard(), auth.NewSessions(q, time.Hour), auth.CookieOptions{AllowedOrigins: []string{testOrigin}}),
		Users:      user.NewHandler(user.NewService(q)),
		Deliveries: delivery.NewHandler(deliveries),
		Live:       delivery.NewLiveHandler(deliveries, live, tokens),
		Ready:      func(r *http.Request) error { return pool.Ping(r.Context()) },
		Options:    opts,
	})

	token := func(name, role string) string {
		u, err := q.CreateUser(context.Background(), store.CreateUserParams{
			Name: name, Email: strings.ToLower(name) + "@example.com", PasswordHash: "x", Role: role,
		})
		require.NoError(t, err)
		tok, err := tokens.Issue(auth.Claims{UserID: u.ID, Role: role})
		require.NoError(t, err)
		return tok
	}
	return &testAPI{
		t: t, h: h, q: q, broker: broker,
		admin:   token("Admin", auth.RoleAdmin),
		driverA: token("Ana", auth.RoleDriver),
		driverB: token("Bruno", auth.RoleDriver),
	}
}

func (a *testAPI) do(method, path, token, body string, into any) int {
	a.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "203.0.113.7:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	if into != nil && rec.Code < 300 {
		require.NoError(a.t, json.Unmarshal(rec.Body.Bytes(), into), rec.Body.String())
	}
	return rec.Code
}

// driverID reads the id of a driver from the admin listing.
func (a *testAPI) driverID(name string) int64 {
	var drivers []user.User
	require.Equal(a.t, http.StatusOK, a.do(http.MethodGet, "/drivers", a.admin, "", &drivers))
	for _, d := range drivers {
		if d.Name == name {
			return d.ID
		}
	}
	a.t.Fatalf("driver %s not found", name)
	return 0
}

func TestIntegration_DriversOnlyReachTheirOwnDeliveries(t *testing.T) {
	a := newAPI(t, Options{})
	var d delivery.Delivery
	body := fmt.Sprintf(`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com","address":"Rua A, 10","driver_id":%d}`, a.driverID("Ana"))
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.admin, body, &d))
	path := fmt.Sprintf("/deliveries/%d", d.ID)

	// Driver B: the delivery of driver A does not exist for him.
	var list []delivery.Delivery
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/me/deliveries", a.driverB, "", &list))
	assert.Empty(t, list)
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodGet, path+"/events", a.driverB, "", nil))
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodPost, path+"/events", a.driverB, `{"status":"picked_up"}`, nil))
	assert.Equal(t, http.StatusForbidden, a.do(http.MethodGet, path, a.driverB, "", nil), "admin routes stay closed to drivers")
	assert.Equal(t, http.StatusForbidden, a.do(http.MethodPatch, path, a.driverB, `{"driver_id":1}`, nil))

	// Driver A works normally.
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/me/deliveries", a.driverA, "", &list))
	require.Len(t, list, 1)
	assert.Equal(t, http.StatusCreated, a.do(http.MethodPost, path+"/events", a.driverA, `{"status":"picked_up"}`, nil))
	assert.Equal(t, http.StatusConflict, a.do(http.MethodPost, path+"/events", a.driverA, `{"status":"picked_up"}`, nil), "repeating an event")
	assert.Equal(t, http.StatusForbidden, a.do(http.MethodGet, "/me/deliveries", a.admin, "", nil), "/me is for drivers")

	// Without a token nothing but the public routes answer.
	assert.Equal(t, http.StatusUnauthorized, a.do(http.MethodGet, "/me/deliveries", "", "", nil))
	assert.Equal(t, http.StatusUnauthorized, a.do(http.MethodPost, path+"/events", "", `{"status":"in_transit"}`, nil))
}

func TestIntegration_PublicTracking(t *testing.T) {
	a := newAPI(t, Options{TrackingRateLimit: 5})
	var d delivery.Delivery
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.admin,
		`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com","address":"Rua A, 10"}`, &d))

	req := httptest.NewRequest(http.MethodGet, "/public/tracking/"+d.TrackingCode, nil)
	req.RemoteAddr = "203.0.113.7:1234"
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	for _, secret := range []string{"Souza", "maria@example.com", "Rua A", `"id"`} {
		assert.NotContains(t, rec.Body.String(), secret)
	}

	assert.Equal(t, http.StatusNotFound, a.do(http.MethodGet, "/public/tracking/RSAAAAAAAAAA", "", "", nil))
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodGet, "/public/tracking/1", "", "", nil), "internal ids are not codes")
	for range 2 {
		a.do(http.MethodGet, "/public/tracking/RSBBBBBBBBBB", "", "", nil)
	}
	assert.Equal(t, http.StatusTooManyRequests, a.do(http.MethodGet, "/public/tracking/"+d.TrackingCode, "", "", nil),
		"guessing codes hits the tracking rate limit")
}

func TestIntegration_IdempotencyKeyHeader(t *testing.T) {
	a := newAPI(t, Options{})
	body := `{"recipient_name":"Maria","recipient_email":"maria@example.com","address":"Rua A, 10"}`
	post := func(key, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/deliveries", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.7:1234"
		req.Header.Set("Authorization", "Bearer "+a.admin)
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		a.h.ServeHTTP(rec, req)
		return rec
	}

	first := post("abc-123", body)
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
	assert.Empty(t, first.Header().Get("Idempotent-Replayed"))

	retry := post("abc-123", body)
	require.Equal(t, http.StatusCreated, retry.Code)
	assert.Equal(t, "true", retry.Header().Get("Idempotent-Replayed"))
	assert.JSONEq(t, first.Body.String(), retry.Body.String())

	other := post("abc-123", strings.Replace(body, "Rua A", "Rua B", 1))
	assert.Equal(t, http.StatusUnprocessableEntity, other.Code)

	var list []delivery.Delivery
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/deliveries", a.admin, "", &list))
	assert.Len(t, list, 1)
}

func TestIntegration_RefreshTokenRotation(t *testing.T) {
	a := newAPI(t, Options{})
	hash, err := auth.HashPassword("senha-da-carla")
	require.NoError(t, err)
	_, err = a.q.CreateUser(context.Background(), store.CreateUserParams{
		Name: "Carla", Email: "carla@example.com", PasswordHash: hash, Role: auth.RoleDriver,
	})
	require.NoError(t, err)

	send := func(path string, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.RemoteAddr = "203.0.113.7:1234"
		req.Header.Set("Origin", testOrigin)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		a.h.ServeHTTP(rec, req)
		return rec
	}
	cookieOf := func(rec *httptest.ResponseRecorder) *http.Cookie {
		for _, c := range rec.Result().Cookies() {
			if c.Name == auth.RefreshCookie {
				return c
			}
		}
		t.Fatalf("no refresh cookie: %d %s", rec.Code, rec.Body.String())
		return nil
	}

	rec := send("/auth/login", nil, `{"email":"carla@example.com","password":"senha-da-carla"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	first := cookieOf(rec)

	rec = send("/auth/refresh", first, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusOK, a.do(http.MethodGet, "/me/deliveries", resp.Token, "", nil),
		"the new access token works")
	second := cookieOf(rec)

	// Replaying the first token revokes the session, including the second token.
	assert.Equal(t, http.StatusUnauthorized, send("/auth/refresh", first, "").Code)
	assert.Equal(t, http.StatusUnauthorized, send("/auth/refresh", second, "").Code)

	// A new login starts a fresh session, and logout ends it.
	third := cookieOf(send("/auth/login", nil, `{"email":"carla@example.com","password":"senha-da-carla"}`))
	assert.Equal(t, http.StatusNoContent, send("/auth/logout", third, "").Code)
	assert.Equal(t, http.StatusUnauthorized, send("/auth/refresh", third, "").Code)
}

func TestIntegration_LiveUpdates(t *testing.T) {
	a := newAPI(t, Options{})
	srv := httptest.NewServer(a.h)
	t.Cleanup(srv.Close)
	base := "ws" + strings.TrimPrefix(srv.URL, "http")
	// dial returns the HTTP status of the handshake.
	dial := func(path string) (*websocket.Conn, int, error) {
		c, res, err := websocket.Dial(t.Context(), base+path, &websocket.DialOptions{
			HTTPHeader: http.Header{"Origin": {testOrigin}},
		})
		if c != nil {
			t.Cleanup(func() { _ = c.CloseNow() })
		}
		status := 0
		if res != nil {
			status = res.StatusCode
			if res.Body != nil {
				_ = res.Body.Close()
			}
		}
		return c, status, err
	}
	read := func(c *websocket.Conn) (string, error) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		_, msg, err := c.Read(ctx)
		return string(msg), err
	}

	var d delivery.Delivery
	body := fmt.Sprintf(`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com","address":"Rua A, 10","driver_id":%d}`,
		a.driverID("Ana"))
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.admin, body, &d))

	_, status, err := dial("/public/tracking/RSAAAAAAAAAA/live")
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, status, "unknown codes are refused before the upgrade")

	driverPanel, _, err := dial("/live/deliveries")
	require.NoError(t, err)
	require.NoError(t, driverPanel.Write(t.Context(), websocket.MessageText, []byte(`{"token":"`+a.driverA+`"}`)))
	_, err = read(driverPanel)
	assert.Equal(t, realtime.StatusUnauthorized, websocket.CloseStatus(err), "the panel stream is for admins only")

	public, _, err := dial("/public/tracking/" + d.TrackingCode + "/live")
	require.NoError(t, err)
	panel, _, err := dial("/live/deliveries")
	require.NoError(t, err)
	require.NoError(t, panel.Write(t.Context(), websocket.MessageText, []byte(`{"token":"`+a.admin+`"}`)))
	require.Eventually(t, func() bool { return a.broker.subs.Load() == 2 }, 5*time.Second, 10*time.Millisecond)

	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, fmt.Sprintf("/deliveries/%d/events", d.ID), a.driverA,
		`{"status":"picked_up"}`, nil))

	msg, err := read(public)
	require.NoError(t, err)
	var tr delivery.Tracking
	require.NoError(t, json.Unmarshal([]byte(msg), &tr))
	assert.Equal(t, delivery.StatusPickedUp, tr.Status)
	for _, secret := range []string{"Souza", "maria@example.com", "Rua A"} {
		assert.NotContains(t, msg, secret)
	}

	msg, err = read(panel)
	require.NoError(t, err)
	assert.JSONEq(t, fmt.Sprintf(`{"delivery_id":%d,"status":"picked_up"}`, d.ID), msg)
}
