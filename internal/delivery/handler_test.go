package delivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRouter() http.Handler {
	h := NewHandler(NewService(newFakeStore()))
	r := chi.NewRouter()
	r.Get("/deliveries", h.List)
	r.Post("/deliveries", h.Create)
	r.Get("/deliveries/{id}", h.Get)
	r.Patch("/deliveries/{id}", h.Update)
	return r
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestHandler_CreateGetUpdate(t *testing.T) {
	r := newTestRouter()

	rec := do(t, r, http.MethodPost, "/deliveries", `{"recipient_name":"Maria","recipient_email":"maria@example.com","address":"Rua A, 10"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created Delivery
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&created))
	assert.Equal(t, StatusPending, created.Status)

	rec = do(t, r, http.MethodGet, "/deliveries/1", "")
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = do(t, r, http.MethodPatch, "/deliveries/1", `{"address":"Rua B, 20"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"address":"Rua B, 20"`)

	rec = do(t, r, http.MethodGet, "/deliveries?status=pending", "")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_Errors(t *testing.T) {
	r := newTestRouter()

	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"invalid input", http.MethodPost, "/deliveries", `{"recipient_name":"","recipient_email":"x","address":""}`, http.StatusUnprocessableEntity},
		{"malformed json", http.MethodPost, "/deliveries", `{`, http.StatusBadRequest},
		{"status is not editable", http.MethodPatch, "/deliveries/1", `{"status":"delivered"}`, http.StatusBadRequest},
		{"bad id", http.MethodGet, "/deliveries/abc", "", http.StatusBadRequest},
		{"missing", http.MethodGet, "/deliveries/99", "", http.StatusNotFound},
		{"bad status filter", http.MethodGet, "/deliveries?status=lost", "", http.StatusUnprocessableEntity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, r, tc.method, tc.path, tc.body)
			assert.Equal(t, tc.want, rec.Code, rec.Body.String())
		})
	}
}
