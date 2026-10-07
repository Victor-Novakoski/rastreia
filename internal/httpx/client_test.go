package httpx

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientKey(t *testing.T) {
	for remote, want := range map[string]string{
		"203.0.113.7:5000":                   "203.0.113.7",
		"[::ffff:203.0.113.7]:5000":          "203.0.113.7",
		"[2001:db8:1:2:aaaa:bbbb:cccc:1]:80": "2001:db8:1:2::/64",
		"[2001:db8:1:2::9]:80":               "2001:db8:1:2::/64",
		"[2001:db8:1:3::9]:80":               "2001:db8:1:3::/64",
		"[fe80::1%eth0]:80":                  "fe80::/64",
		"not an address":                     "not an address",
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		assert.Equal(t, want, ClientKey(r), remote)
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "[2001:db8::1]:443"
	assert.Equal(t, "2001:db8::1", ClientIP(r))
	r.RemoteAddr = "203.0.113.7"
	assert.Equal(t, "203.0.113.7", ClientIP(r), "no port")
}
