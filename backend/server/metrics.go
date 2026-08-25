package server

import (
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// newMetricsHandler keeps the server-local registry isolated while folding in
// package-global instrumentation that is registered elsewhere in Bytebase.
func newMetricsHandler(registry *prometheus.Registry) http.Handler {
	return promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(
			prometheus.Gatherers{registry, prometheus.DefaultGatherer},
			promhttp.HandlerOpts{ErrorHandling: promhttp.HTTPErrorOnError},
		),
	)
}

func metricsAccessHandler(remoteAccess bool, next http.Handler) http.Handler {
	if remoteAccess {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if addr := net.ParseIP(host); addr == nil || !addr.IsLoopback() {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
