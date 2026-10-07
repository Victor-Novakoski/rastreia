package httpx

import (
	"net"
	"net/http"
	"net/netip"
)

// ClientIP is the IP of r.RemoteAddr without the port. Behind the load
// balancer the server's trustedProxy middleware has already put the real
// client IP there.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ClientKey is what the per-client limits count by: the IPv4 address, or the
// /64 network of an IPv6 one. An IPv6 client usually gets a whole /64 and
// can switch addresses inside it at will, like a home behind one IPv4.
func ClientKey(r *http.Request) string {
	ip := ClientIP(r)
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}
	p, err := addr.WithZone("").Prefix(64)
	if err != nil {
		return ip
	}
	return p.String()
}
