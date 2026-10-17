package routes

import (
	"net/http"
	"net/netip"

	"github.com/AlexGalhardo/vukem/web"
)

// IPKey keys rate limits by client address: IPv4 as is, IPv6 by its /64
// (one subscriber usually owns a whole /64, so per-address keys are free to
// rotate).
func IPKey(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "ip:" + ip
	}

	if addr.Is4() || addr.Is4In6() {
		return "ip:" + addr.Unmap().String()
	}

	prefix, err := addr.Prefix(64)
	if err != nil {
		return "ip:" + ip
	}

	return "ip6:" + prefix.String()
}

// byClientIP is the ratelimit key func over IPKey (c.IP honours
// TRUSTED_PROXIES / CLIENT_IP_HEADER).
func byClientIP(r *http.Request) string {
	return IPKey(web.NewCtx(nil, r, web.ServicesFrom(r.Context())).IP())
}
