// Package apperr defines the errors services return so handlers can map
// them to HTTP status codes without knowing about the database.
package apperr

import (
	"errors"
	"strings"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// ValidationError lists every invalid field of a request.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for field, msg := range e.Fields {
		parts = append(parts, field+": "+msg)
	}
	return "invalid input: " + strings.Join(parts, ", ")
}

// Validator collects field errors; call Err at the end.
type Validator map[string]string

func (v Validator) Check(ok bool, field, msg string) {
	if !ok {
		if _, exists := v[field]; !exists {
			v[field] = msg
		}
	}
}

func (v Validator) Err() error {
	if len(v) == 0 {
		return nil
	}
	return &ValidationError{Fields: v}
}
