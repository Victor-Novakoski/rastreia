package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/auth"
)

// captureLogs sends slog's output to a buffer until the test ends and
// returns a function that decodes the lines written so far.
func captureLogs(t *testing.T) func() []map[string]any {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return func() []map[string]any {
		var lines []map[string]any
		for _, l := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if l == "" {
				continue
			}
			var m map[string]any
			require.NoError(t, json.Unmarshal([]byte(l), &m), l)
			lines = append(lines, m)
		}
		return lines
	}
}

func lastRequest(t *testing.T, lines []map[string]any) map[string]any {
	t.Helper()
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i]["msg"] == "request" || lines[i]["msg"] == "request denied" {
			return lines[i]
		}
	}
	t.Fatal("no request line logged")
	return nil
}

func TestRequestLog(t *testing.T) {
	logs := captureLogs(t)
	h := newTestServer(Options{RateLimit: 5})
	tokens := auth.NewTokens("test-secret-with-at-least-32-characters", time.Hour)
	driver, err := tokens.Issue(auth.Claims{UserID: 7, Role: auth.RoleDriver, CarrierID: 3})
	require.NoError(t, err)

	get(h, "/health", nil)
	line := lastRequest(t, logs())
	assert.Equal(t, "INFO", line["level"])
	assert.Equal(t, "request", line["msg"])
	assert.Equal(t, "GET", line["method"])
	assert.Equal(t, "/health", line["route"])
	assert.EqualValues(t, http.StatusOK, line["status"])
	assert.Equal(t, "203.0.113.7", line["ip"])
	assert.NotEmpty(t, line["request_id"])

	get(h, "/deliveries?q=Maria+Souza", nil)
	line = lastRequest(t, logs())
	assert.Equal(t, "WARN", line["level"])
	assert.Equal(t, "request denied", line["msg"])
	assert.EqualValues(t, http.StatusUnauthorized, line["status"])
	assert.Equal(t, "missing token", line["reason"])
	assert.Equal(t, "/deliveries", line["route"])

	get(h, "/deliveries", map[string]string{"Authorization": "Bearer not-a-jwt"})
	assert.Equal(t, "invalid token", lastRequest(t, logs())["reason"])

	get(h, "/deliveries", map[string]string{"Authorization": "Bearer " + driver})
	line = lastRequest(t, logs())
	assert.EqualValues(t, http.StatusForbidden, line["status"])
	assert.Equal(t, "role driver not allowed", line["reason"])
	assert.EqualValues(t, 7, line["user_id"])
	assert.EqualValues(t, 3, line["carrier_id"])

	// 0 and 1 are not in the alphabet, so the code is refused before the database.
	get(h, "/public/tracking/RS0000000001", nil)
	line = lastRequest(t, logs())
	assert.Equal(t, "/public/tracking/{code}", line["route"])
	assert.EqualValues(t, http.StatusNotFound, line["status"])

	rec := get(h, "/health", nil)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	line = lastRequest(t, logs())
	assert.Equal(t, "WARN", line["level"])
	assert.Equal(t, "rate limit global", line["reason"])

	all, _ := json.Marshal(logs())
	assert.NotContains(t, string(all), "Maria", "the query string is not logged")
	assert.NotContains(t, string(all), "RS0000000001", "tracking codes are not logged")
	assert.NotContains(t, string(all), driver, "tokens are not logged")
}

func TestRecoverer(t *testing.T) {
	logs := captureLogs(t)
	h := requestLog(recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"error":"internal error"}`, rec.Body.String())
	lines := logs()
	require.Len(t, lines, 2)
	assert.Equal(t, "panic", lines[0]["msg"])
	assert.Equal(t, "boom", lines[0]["err"])
	assert.Contains(t, lines[0]["stack"], "log_test.go")
	assert.Equal(t, "ERROR", lines[1]["level"])
	assert.EqualValues(t, http.StatusInternalServerError, lines[1]["status"])
}
