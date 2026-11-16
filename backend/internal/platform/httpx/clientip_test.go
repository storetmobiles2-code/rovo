package httpx

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name, remote, xff, want string
	}{
		{"direct public peer", "203.0.113.7:4411", "", "203.0.113.7"},
		{"public peer cannot spoof with a header", "203.0.113.7:4411", "198.51.100.1", "203.0.113.7"},
		{"behind our proxy", "10.0.0.5:5000", "198.51.100.1", "198.51.100.1"},
		{"proxy chain: the entry our proxy appended wins", "127.0.0.1:5000", "1.2.3.4, 198.51.100.9", "198.51.100.9"},
		{"private peer without a header", "172.18.0.3:5000", "", "172.18.0.3"},
		{"garbage header falls back to the peer", "10.0.0.5:5000", "not-an-ip", "10.0.0.5"},
		{"ipv6 peer", "[2001:db8::1]:443", "", "2001:db8::1"},
		{"ipv4-mapped ipv6 is unmapped", "[::ffff:203.0.113.7]:443", "", "203.0.113.7"},
		{"unparseable peer", "pipe", "198.51.100.1", ""},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := ClientIP(r); got != c.want {
			t.Errorf("%s: ClientIP = %q, want %q", c.name, got, c.want)
		}
	}
}
