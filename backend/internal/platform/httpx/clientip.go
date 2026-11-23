package httpx

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIP returns the caller's IP for rate limiting and audit. The API is only reachable through
// the edge proxy (Caddy / CDN, RV-001), so when the direct peer is a private or loopback address
// the right-most X-Forwarded-For entry (the one our own proxy appended) is used. A public peer is
// taken as-is and its forwarded header is ignored, so a direct caller cannot spoof an address.
// It returns "" when no valid address is found.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return ""
	}
	peer = peer.Unmap()
	if peer.IsPrivate() || peer.IsLoopback() {
		xff := r.Header.Get("X-Forwarded-For")
		if i := strings.LastIndexByte(xff, ','); i >= 0 {
			xff = xff[i+1:]
		}
		if fwd, err := netip.ParseAddr(strings.TrimSpace(xff)); err == nil {
			return fwd.Unmap().String()
		}
	}
	return peer.String()
}
