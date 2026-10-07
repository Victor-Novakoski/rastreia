package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Victor-Novakoski/rastreia/internal/httpx"
)

// requestLog writes one JSON line per request through slog, like every other
// log of the API. Denied requests (401, 403 and 429) go out as warnings with
// the message "request denied", so they can be searched and turned into
// alerts (SECURITY.md #26); errors (5xx) go out as errors.
//
// It logs the route pattern (/public/tracking/{code}) instead of the path
// and never the query string, so tracking codes and searches, which may
// carry a recipient's name, stay out of the logs.
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx, extra := httpx.WithLogAttrs(r.Context())
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		r = r.WithContext(ctx)
		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK // nothing written: net/http answers 200
		}
		level, msg := slog.LevelInfo, "request"
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusTooManyRequests:
			level, msg = slog.LevelWarn, "request denied"
		}
		ip, _ := remoteIP(r)
		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("route", routePattern(r)),
			slog.Int("status", status),
			slog.Int("bytes", ww.BytesWritten()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("ip", ip),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		}
		slog.LogAttrs(r.Context(), level, msg, append(attrs, extra.Attrs()...)...)
	})
}

// routePattern is the route that answered, or the path (cut short) when no
// route matched, which is what scanners probing for other apps hit.
func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if p := rc.RoutePattern(); p != "" {
			return p
		}
	}
	const maxPath = 100
	if p := r.URL.Path; len(p) > maxPath {
		return p[:maxPath]
	}
	return r.URL.Path
}

// recoverer turns a panic into a 500 and logs it with the stack, in JSON
// like the rest, instead of chi's Recoverer, which prints text to stderr.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec) // the client went away; net/http handles it
			}
			slog.Error("panic", "err", rec, "stack", string(debug.Stack()),
				"request_id", middleware.GetReqID(r.Context()))
			if r.Header.Get("Connection") != "Upgrade" {
				httpx.Error(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
