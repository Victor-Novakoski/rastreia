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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/realtime"
	"github.com/Victor-Novakoski/rastreia/internal/route"
	"github.com/Victor-Novakoski/rastreia/internal/store"
	"github.com/Victor-Novakoski/rastreia/internal/testdb"
	"github.com/Victor-Novakoski/rastreia/internal/testredis"
	"github.com/Victor-Novakoski/rastreia/internal/user"
)

const testOrigin = "http://localhost:5173"

// testAPI is the real router on a real database, with one token per user.
// owner runs the carrier with drivers A and B; rival runs another carrier.
type testAPI struct {
	t         *testing.T
	h         http.Handler
	q         *store.Queries
	carrierID int64
	owner     string
	driverA   string
	driverB   string
	rival     string
	broker    *countingBroker
	pool      *pgxpool.Pool
	tokens    *auth.Tokens
	opts      Options
}

// newInstance builds the router as cmd/api does, on a shared database.
func newInstance(t *testing.T, pool *pgxpool.Pool, tokens *auth.Tokens, opts Options) (http.Handler, *countingBroker) {
	var (
		b     realtime.Broker = realtime.NewLocal()
		guard auth.Guard      = auth.NewLoginGuard()
	)
	if opts.Redis != nil {
		rb, err := realtime.NewRedis(t.Context(), opts.Redis)
		require.NoError(t, err)
		b, guard = rb, auth.NewRedisGuard(opts.Redis)
	}
	broker := &countingBroker{Broker: b}
	q := store.New(pool)
	deliveries := delivery.NewService(delivery.NewPGStore(pool)).WithPublisher(broker)
	live := realtime.NewServer(t.Context(), broker, realtime.Options{Origins: []string{testOrigin}})
	authHandler := auth.NewHandler(q, tokens, guard, auth.NewSessions(q, time.Hour), auth.CookieOptions{AllowedOrigins: []string{testOrigin}})
	return New(Deps{
		Tokens:     tokens,
		Auth:       authHandler,
		Users:      user.NewHandler(user.NewService(q), authHandler),
		Deliveries: delivery.NewHandler(deliveries),
		Routes:     route.NewHandler(route.NewService(route.NewPGStore(pool), deliveries)),
		Live:       delivery.NewLiveHandler(deliveries, live, tokens),
		Ready:      func(r *http.Request) error { return pool.Ping(r.Context()) },
		Options:    opts,
	}), broker
}

// countingBroker tells tests when a WebSocket is listening, so a change is
// not published before anyone subscribed.
type countingBroker struct {
	realtime.Broker
	subs atomic.Int32
}

func (b *countingBroker) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	defer b.subs.Add(1)
	return b.Broker.Subscribe(ctx, topic)
}

// newAPI starts one API instance. With opts.Redis set, live updates and
// login lockouts go through Redis too, as in production.
func newAPI(t *testing.T, opts Options) *testAPI {
	pool := testdb.New(t)
	q := store.New(pool)
	tokens := auth.NewTokens("test-secret-with-at-least-32-characters", time.Hour)
	h, broker := newInstance(t, pool, tokens, opts)

	carrierID, rivalID := testdb.Carrier(t, pool), testdb.Carrier(t, pool)
	token := func(carrierID int64, name, role string) string {
		u, err := q.CreateUser(context.Background(), store.CreateUserParams{
			CarrierID: carrierID, Name: name, Email: strings.ToLower(name) + "@example.com", PasswordHash: "x", Role: role,
		})
		require.NoError(t, err)
		tok, err := tokens.Issue(auth.Claims{UserID: u.ID, Role: role, CarrierID: carrierID})
		require.NoError(t, err)
		return tok
	}
	return &testAPI{
		t: t, h: h, q: q, broker: broker, pool: pool, tokens: tokens, opts: opts, carrierID: carrierID,
		owner:   token(carrierID, "Dona", auth.RoleCarrier),
		driverA: token(carrierID, "Ana", auth.RoleDriver),
		driverB: token(carrierID, "Bruno", auth.RoleDriver),
		rival:   token(rivalID, "Rival", auth.RoleCarrier),
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

// driverID reads the id of a driver from the owner listing.
func (a *testAPI) driverID(name string) int64 {
	var drivers []user.User
	require.Equal(a.t, http.StatusOK, a.do(http.MethodGet, "/drivers", a.owner, "", &drivers))
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
	body := fmt.Sprintf(`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com",`+addressJSON+`,"driver_id":%d}`, a.driverID("Ana"))
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.owner, body, &d))
	path := fmt.Sprintf("/deliveries/%d", d.ID)

	// Driver B: the delivery of driver A does not exist for him.
	var list []delivery.Delivery
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/me/deliveries", a.driverB, "", &list))
	assert.Empty(t, list)
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodGet, path+"/events", a.driverB, "", nil))
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodPost, path+"/events", a.driverB, `{"status":"picked_up"}`, nil))
	assert.Equal(t, http.StatusForbidden, a.do(http.MethodGet, path, a.driverB, "", nil), "carrier routes stay closed to drivers")
	assert.Equal(t, http.StatusForbidden, a.do(http.MethodPatch, path, a.driverB, `{"driver_id":1}`, nil))

	// Driver A works normally.
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/me/deliveries", a.driverA, "", &list))
	require.Len(t, list, 1)
	assert.Equal(t, http.StatusCreated, a.do(http.MethodPost, path+"/events", a.driverA, `{"status":"picked_up"}`, nil))
	assert.Equal(t, http.StatusConflict, a.do(http.MethodPost, path+"/events", a.driverA, `{"status":"picked_up"}`, nil), "repeating an event")
	assert.Equal(t, http.StatusForbidden, a.do(http.MethodGet, "/me/deliveries", a.owner, "", nil), "/me is for drivers")

	// Without a token nothing but the public routes answer.
	assert.Equal(t, http.StatusUnauthorized, a.do(http.MethodGet, "/me/deliveries", "", "", nil))
	assert.Equal(t, http.StatusUnauthorized, a.do(http.MethodPost, path+"/events", "", `{"status":"in_transit"}`, nil))
}

func TestIntegration_PublicTracking(t *testing.T) {
	a := newAPI(t, Options{TrackingRateLimit: 5})
	var d delivery.Delivery
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.owner,
		`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com",`+addressJSON+`}`, &d))

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
	body := `{"recipient_name":"Maria","recipient_email":"maria@example.com",` + addressJSON + `}`
	post := func(key, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/deliveries", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.7:1234"
		req.Header.Set("Authorization", "Bearer "+a.owner)
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

	other := post("abc-123", strings.Replace(body, `"number":"10"`, `"number":"20"`, 1))
	assert.Equal(t, http.StatusUnprocessableEntity, other.Code)

	var list []delivery.Delivery
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/deliveries", a.owner, "", &list))
	assert.Len(t, list, 1)
}

func TestIntegration_RefreshTokenRotation(t *testing.T) {
	a := newAPI(t, Options{})
	hash, err := auth.HashPassword("senha-da-carla")
	require.NoError(t, err)
	_, err = a.q.CreateUser(context.Background(), store.CreateUserParams{
		CarrierID: a.carrierID, Name: "Carla", Email: "carla@example.com", PasswordHash: hash, Role: auth.RoleDriver,
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
	dial := func(path string) (*websocket.Conn, int, error) { return dialWS(t, srv.URL, path) }
	read := func(c *websocket.Conn) (string, error) { return readWS(t, c) }

	var d delivery.Delivery
	body := fmt.Sprintf(`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com",`+addressJSON+`,"driver_id":%d}`,
		a.driverID("Ana"))
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.owner, body, &d))

	_, status, err := dial("/public/tracking/RSAAAAAAAAAA/live")
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, status, "unknown codes are refused before the upgrade")

	driverPanel, _, err := dial("/live/deliveries")
	require.NoError(t, err)
	require.NoError(t, driverPanel.Write(t.Context(), websocket.MessageText, []byte(`{"token":"`+a.driverA+`"}`)))
	_, err = read(driverPanel)
	assert.Equal(t, realtime.StatusUnauthorized, websocket.CloseStatus(err), "the panel stream is for carriers only")

	public, _, err := dial("/public/tracking/" + d.TrackingCode + "/live")
	require.NoError(t, err)
	panel, _, err := dial("/live/deliveries")
	require.NoError(t, err)
	require.NoError(t, panel.Write(t.Context(), websocket.MessageText, []byte(`{"token":"`+a.owner+`"}`)))
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

func TestIntegration_CarriersAreIsolated(t *testing.T) {
	a := newAPI(t, Options{})
	var d delivery.Delivery
	body := fmt.Sprintf(`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com",`+addressJSON+`,"driver_id":%d}`, a.driverID("Ana"))
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.owner, body, &d))
	path := fmt.Sprintf("/deliveries/%d", d.ID)

	var list []delivery.Delivery
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/deliveries", a.rival, "", &list))
	assert.Empty(t, list)
	var drivers []user.User
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/drivers", a.rival, "", &drivers))
	assert.Empty(t, drivers)
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodGet, path, a.rival, "", nil))
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodPatch, path, a.rival, `{"number":"20"}`, nil))
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodGet, path+"/events", a.rival, "", nil))
	assert.Equal(t, http.StatusNotFound, a.do(http.MethodPost, path+"/events", a.rival, `{"status":"picked_up"}`, nil))
	assert.Equal(t, http.StatusUnprocessableEntity, a.do(http.MethodPost, "/deliveries", a.rival, body, nil),
		"another carrier's driver cannot be assigned")

	var sum delivery.Summary
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/summary", a.rival, "", &sum))
	assert.Zero(t, sum.ByStatus[delivery.StatusPending])
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/summary", a.owner, "", &sum))
	assert.Equal(t, int64(1), sum.ByStatus[delivery.StatusPending])
	assert.Equal(t, http.StatusForbidden, a.do(http.MethodGet, "/summary", a.driverA, "", nil))
}

func TestIntegration_SignUp(t *testing.T) {
	a := newAPI(t, Options{})
	signUp := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/auth/signup", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.7:1234"
		rec := httptest.NewRecorder()
		a.h.ServeHTTP(rec, req)
		return rec
	}
	body := `{"carrier_name":"Expresso Sul","document":"11.222.333/0001-81","name":"Carla","email":"carla@example.com","password":"transporte-forte"}`

	rec := signUp(body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp struct {
		Token string `json:"token"`
		Role  string `json:"role"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, auth.RoleCarrier, resp.Role)
	assert.Contains(t, rec.Header().Get("Set-Cookie"), auth.RefreshCookie, "signing up also logs in")

	var me user.Me
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/me", resp.Token, "", &me))
	assert.Equal(t, "Expresso Sul", me.Carrier.Name)
	assert.Equal(t, "Carla", me.Name)
	var list []delivery.Delivery
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/deliveries", resp.Token, "", &list))
	assert.Empty(t, list, "a new carrier starts empty")

	assert.Equal(t, http.StatusConflict, signUp(body).Code)
	assert.Equal(t, http.StatusUnprocessableEntity, signUp(`{"carrier_name":"","name":"X","email":"x@example.com","password":"transporte-forte"}`).Code)
	assert.Equal(t, http.StatusBadRequest, signUp(`{"carrier_name":"X","role":"carrier"}`).Code, "unknown fields are refused")
	assert.Equal(t, http.StatusUnauthorized, a.do(http.MethodGet, "/me", "", "", nil))
}

// dialWS opens a WebSocket from the front's origin and returns the HTTP
// status of the handshake.
func dialWS(t *testing.T, serverURL, path string) (*websocket.Conn, int, error) {
	t.Helper()
	c, res, err := websocket.Dial(t.Context(), "ws"+strings.TrimPrefix(serverURL, "http")+path, &websocket.DialOptions{
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

func readWS(t *testing.T, c *websocket.Conn) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	_, msg, err := c.Read(ctx)
	return string(msg), err
}

// Two API instances on the same database and Redis, as behind a load
// balancer: a change made on one reaches a browser on the other, and
// limits and lockouts count requests to both.
func TestIntegration_InstancesShareRedis(t *testing.T) {
	a := newAPI(t, Options{Redis: testredis.New(t), TrackingRateLimit: 4})
	other, otherBroker := newInstance(t, a.pool, a.tokens, a.opts)
	srvB := httptest.NewServer(other)
	t.Cleanup(srvB.Close)

	var d delivery.Delivery
	body := fmt.Sprintf(`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com",`+addressJSON+`,"driver_id":%d}`,
		a.driverID("Ana"))
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.owner, body, &d))

	public, _, err := dialWS(t, srvB.URL, "/public/tracking/"+d.TrackingCode+"/live")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return otherBroker.subs.Load() == 1 }, 5*time.Second, 10*time.Millisecond)
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, fmt.Sprintf("/deliveries/%d/events", d.ID), a.driverA,
		`{"status":"picked_up"}`, nil))
	msg, err := readWS(t, public)
	require.NoError(t, err)
	assert.Contains(t, msg, `"status":"picked_up"`, "a change made on A reaches the browser on B")

	send := func(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.RemoteAddr = "198.51.100.20:1234"
		req.Header.Set("Origin", testOrigin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	tracking := "/public/tracking/" + d.TrackingCode
	for _, h := range []http.Handler{a.h, a.h, other, other} {
		require.Equal(t, http.StatusOK, send(h, http.MethodGet, tracking, "").Code)
	}
	assert.Equal(t, http.StatusTooManyRequests, send(other, http.MethodGet, tracking, "").Code,
		"the tracking limit counts lookups on both instances")

	login := `{"email":"ana@example.com","password":"wrong"}`
	for range 5 {
		require.Equal(t, http.StatusUnauthorized, send(a.h, http.MethodPost, "/auth/login", login).Code)
	}
	rec := send(other, http.MethodPost, "/auth/login", login)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed attempts", "an e-mail locked on A is locked on B")
}

// The route is the driver's own: carriers have none, and each driver only
// reaches packages of their carrier.
func TestIntegration_Route(t *testing.T) {
	a := newAPI(t, Options{})
	var d delivery.Delivery
	require.Equal(t, http.StatusCreated, a.do(http.MethodPost, "/deliveries", a.owner,
		`{"recipient_name":"Maria Souza","recipient_email":"maria@example.com",`+addressJSON+`}`, &d))

	var r route.Route
	assert.Equal(t, http.StatusForbidden, a.do(http.MethodGet, "/me/route", a.owner, "", nil))
	assert.Equal(t, http.StatusUnauthorized, a.do(http.MethodGet, "/me/route", "", "", nil))

	require.Equal(t, http.StatusOK, a.do(http.MethodPost, "/me/route/deliveries", a.driverA, `{"code":"`+d.TrackingCode+`"}`, &r))
	require.Len(t, r.Stops, 1)
	assert.Equal(t, 1, r.Stops[0].Packages[0].Position)
	assert.Equal(t, "10", r.Stops[0].Packages[0].Number, "the position does not hide the street number")
	assert.Equal(t, http.StatusConflict, a.do(http.MethodPost, "/me/route/deliveries", a.driverB, `{"code":"`+d.TrackingCode+`"}`, nil),
		"the first driver to scan takes the package")

	assert.Equal(t, http.StatusOK, a.do(http.MethodPost, "/me/route/optimize", a.driverA, `{"latitude":-23.5,"longitude":-46.6}`, &r))
	assert.Equal(t, http.StatusUnprocessableEntity, a.do(http.MethodPost, "/me/route/optimize", a.driverA, `{"latitude":-23.5}`, nil))
	assert.Equal(t, http.StatusOK, a.do(http.MethodPut, "/me/route/order", a.driverA, fmt.Sprintf(`{"delivery_ids":[%d]}`, d.ID), &r))
	assert.Equal(t, http.StatusUnprocessableEntity, a.do(http.MethodPut, "/me/route/order", a.driverA, `{"delivery_ids":[]}`, nil))
	assert.Equal(t, http.StatusOK, a.do(http.MethodDelete, fmt.Sprintf("/me/route/deliveries/%d", d.ID), a.driverA, "", &r))
	assert.Empty(t, r.Stops)
	assert.Equal(t, http.StatusOK, a.do(http.MethodGet, "/me/route", a.driverB, "", &r))
	assert.Empty(t, r.Stops)
}

const addressJSON = `"recipient_phone":"11987654321","postal_code":"01001000","street":"Praça da Sé","number":"10",` +
	`"district":"Sé","city":"São Paulo","state":"SP"`
