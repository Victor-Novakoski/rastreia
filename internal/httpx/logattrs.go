package httpx

import (
	"context"
	"log/slog"
	"sync"
)

type logAttrsKey struct{}

// LogAttrs collects what the handlers learn about a request (who made it,
// why it was denied) for the single log line written when it ends.
type LogAttrs struct {
	mu    sync.Mutex
	attrs []slog.Attr
}

// WithLogAttrs returns a context where AddLogAttrs works, and the collected
// attributes.
func WithLogAttrs(ctx context.Context) (context.Context, *LogAttrs) {
	la := &LogAttrs{}
	return context.WithValue(ctx, logAttrsKey{}, la), la
}

// AddLogAttrs attaches attributes to the request's log line. Outside a
// logged request it does nothing.
func AddLogAttrs(ctx context.Context, attrs ...slog.Attr) {
	if la, ok := ctx.Value(logAttrsKey{}).(*LogAttrs); ok {
		la.mu.Lock()
		la.attrs = append(la.attrs, attrs...)
		la.mu.Unlock()
	}
}

// Attrs returns the attributes added so far.
func (la *LogAttrs) Attrs() []slog.Attr {
	la.mu.Lock()
	defer la.mu.Unlock()
	return append([]slog.Attr(nil), la.attrs...)
}
