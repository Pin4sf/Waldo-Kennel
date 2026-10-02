package httpd

import (
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ownercommand"
	"github.com/go-chi/chi/v5"
	"net"
	"net/http"
	"strings"
)

func mountBridge(r chi.Router, h http.Handler, authority *ownercommand.BridgeAuthority) {
	if h == nil || authority == nil {
		return
	}
	r.Handle("/internal/bridge/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		for name := range req.Header {
			if strings.EqualFold(name, "Origin") {
				notFoundJSON(w, req)
				return
			}
		}
		if !localControlRequest(req) {
			notFoundJSON(w, req)
			return
		}
		http.StripPrefix("/internal/bridge", h).ServeHTTP(w, req)
	}))
}

// Apply the private route boundary before global CORS and proxy-IP rewriting.
// The route must retain JSON 404 and inspect the actual peer, including when a
// forged forwarding header is present. Public API middleware stays unchanged.
func bridgeLocalGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/internal/bridge/") {
			for name := range r.Header {
				if strings.EqualFold(name, "Origin") {
					notFoundJSON(w, r)
					return
				}
			}
			if !localControlRequest(r) {
				notFoundJSON(w, r)
				return
			}
			host, _, e := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(host)
			if e != nil || ip == nil || !ip.IsLoopback() {
				notFoundJSON(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
