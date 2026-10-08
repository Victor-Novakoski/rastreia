// Package httpx holds small helpers shared by the HTTP handlers.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
)

type errorBody struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, errorBody{Error: msg})
}

// Decode reads a JSON body into dst, rejecting unknown fields, bodies over
// 1 MB and text with a NUL character (\u0000), which JSON allows and
// PostgreSQL does not store.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is empty")
		}
		return errors.New("invalid JSON body")
	}
	if hasNUL(reflect.ValueOf(dst)) {
		return errors.New(`text must not contain \u0000`)
	}
	return nil
}

func hasNUL(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return strings.ContainsRune(v.String(), 0)
	case reflect.Pointer, reflect.Interface:
		return !v.IsNil() && hasNUL(v.Elem())
	case reflect.Struct:
		for i := range v.NumField() {
			if hasNUL(v.Field(i)) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if hasNUL(v.Index(i)) {
				return true
			}
		}
	case reflect.Map:
		for it := v.MapRange(); it.Next(); {
			if hasNUL(it.Key()) || hasNUL(it.Value()) {
				return true
			}
		}
	}
	return false
}

type validationBody struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields"`
}

// statusClientClosed is what nginx logs when the client gave up first.
const statusClientClosed = 499

// WriteError maps service errors to HTTP responses. Unknown errors become 500
// without leaking their message.
func WriteError(w http.ResponseWriter, err error) {
	var verr *apperr.ValidationError
	switch {
	case errors.Is(err, context.Canceled):
		// The client went away (closed the page, cancelled the request), so
		// nobody reads the answer and nothing went wrong here.
		w.WriteHeader(statusClientClosed)
	case errors.Is(err, context.DeadlineExceeded):
		Error(w, http.StatusGatewayTimeout, "the request took too long")
	case errors.As(err, &verr):
		JSON(w, http.StatusUnprocessableEntity, validationBody{Error: "invalid input", Fields: verr.Fields})
	case errors.Is(err, apperr.ErrNotFound):
		Error(w, http.StatusNotFound, "not found")
	case errors.Is(err, apperr.ErrConflict):
		Error(w, http.StatusConflict, err.Error())
	default:
		slog.Error("unexpected error", "err", err)
		Error(w, http.StatusInternalServerError, "internal error")
	}
}
