package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
)

func TestDecode(t *testing.T) {
	type address struct {
		Street string `json:"street"`
	}
	type input struct {
		Name    string            `json:"name"`
		Note    *string           `json:"note"`
		Address address           `json:"address"`
		Tags    []string          `json:"tags"`
		Extra   map[string]string `json:"extra"`
	}
	for body, want := range map[string]string{
		`{"name":"Ana","note":"oi","address":{"street":"Rua A"},"tags":["a"],"extra":{"k":"v"}}`: "",
		``:                                    "request body is empty",
		`{"name":`:                            "invalid JSON body",
		`{"nome":"Ana"}`:                      "invalid JSON body",
		`{"name":"Ana\u0000"}`:                `text must not contain \u0000`,
		`{"note":"\u0000"}`:                   `text must not contain \u0000`,
		`{"address":{"street":"Rua\u0000A"}}`: `text must not contain \u0000`,
		`{"tags":["a","\u0000"]}`:             `text must not contain \u0000`,
		`{"extra":{"\u0000":"v"}}`:            `text must not contain \u0000`,
		`{"name":"Ana \\u0000"}`:              "",
	} {
		var in input
		err := Decode(httptest.NewRecorder(), httptest.NewRequest("POST", "/", strings.NewReader(body)), &in)
		if want == "" {
			assert.NoError(t, err, body)
		} else {
			assert.EqualError(t, err, want, body)
		}
	}
}

func TestWriteError(t *testing.T) {
	for _, c := range []struct {
		err    error
		status int
		body   string
	}{
		{&apperr.ValidationError{Fields: map[string]string{"name": "is required"}}, http.StatusUnprocessableEntity, `{"error":"invalid input","fields":{"name":"is required"}}`},
		{fmt.Errorf("get: %w", apperr.ErrNotFound), http.StatusNotFound, `{"error":"not found"}`},
		{fmt.Errorf("%w: taken", apperr.ErrConflict), http.StatusConflict, `{"error":"conflict: taken"}`},
		{fmt.Errorf("query: %w", context.Canceled), statusClientClosed, ``},
		{fmt.Errorf("query: %w", context.DeadlineExceeded), http.StatusGatewayTimeout, `{"error":"the request took too long"}`},
		{errors.New("password authentication failed for user"), http.StatusInternalServerError, `{"error":"internal error"}`},
	} {
		w := httptest.NewRecorder()
		WriteError(w, c.err)
		assert.Equal(t, c.status, w.Code, c.err)
		assert.Equal(t, c.body, strings.TrimSpace(w.Body.String()), c.err)
	}
}
