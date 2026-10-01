// Package httpx holds small helpers shared by the HTTP handlers.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

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

// Decode reads a JSON body into dst, rejecting unknown fields and bodies over 1 MB.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is empty")
		}
		return errors.New("invalid JSON body")
	}
	return nil
}

type validationBody struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields"`
}

// WriteError maps service errors to HTTP responses. Unknown errors become 500
// without leaking their message.
func WriteError(w http.ResponseWriter, err error) {
	var verr *apperr.ValidationError
	switch {
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
