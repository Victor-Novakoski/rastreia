package server

import (
	"net"
	"net/http"
	"strings"
)

// trustedProxy rewrites r.RemoteAddr to the client IP reported by the proxy
// in front of the API. It reads the RIGHTMOST X-Forwarded-For entry: that is
// the one our own load balancer appended. Entries to its left come from the
// client and can be forged, which is why chi's RealIP (leftmost value) is not
// used. Only enable it when the API is reachable solely through that proxy.
func trustedProxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := lastForwardedFor(r); ip != "" {
			r.RemoteAddr = net.JoinHostPort(ip, "0")
		}
		next.ServeHTTP(w, r)
	})
}

func lastForwardedFor(r *http.Request) string {
	values := r.Header.Values("X-Forwarded-For")
	if len(values) == 0 {
		return ""
	}
	parts := strings.Split(values[len(values)-1], ",")
	ip := net.ParseIP(strings.TrimSpace(parts[len(parts)-1]))
	if ip == nil {
		return ""
	}
	return ip.String()
}
