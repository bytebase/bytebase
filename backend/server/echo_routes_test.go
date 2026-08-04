package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/bytebase/bytebase/backend/component/config"
)

func TestSecurityHeadersMiddleware_GA4Sources(t *testing.T) {
	tests := []struct {
		name           string
		saas           bool
		wantGoogleGA4  bool
		wantBaseSource []string
	}{
		{
			name:          "self-host does not allow GA4 sources",
			saas:          false,
			wantGoogleGA4: false,
			wantBaseSource: []string{
				"script-src 'self'",
				"connect-src 'self'",
				"img-src 'self'",
			},
		},
		{
			name:          "SaaS allows GA4 sources",
			saas:          true,
			wantGoogleGA4: true,
			wantBaseSource: []string{
				"script-src 'self'",
				"connect-src 'self'",
				"img-src 'self'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csp := testSecurityHeadersCSP(t, tt.saas)
			for _, want := range tt.wantBaseSource {
				if !strings.Contains(csp, want) {
					t.Errorf("Content-Security-Policy = %q, want to contain %q", csp, want)
				}
			}
			for _, googleSource := range []string{
				"https://www.googletagmanager.com",
				"https://*.google-analytics.com",
				"https://*.analytics.google.com",
				"https://*.googletagmanager.com",
			} {
				if got := strings.Contains(csp, googleSource); got != tt.wantGoogleGA4 {
					t.Errorf("Content-Security-Policy contains %q = %t, want %t; csp=%q", googleSource, got, tt.wantGoogleGA4, csp)
				}
			}
		})
	}
}

func TestSecurityHeadersMiddleware_SaaSAllowsPostHogHosts(t *testing.T) {
	csp := testSecurityHeadersCSP(t, true)
	for _, directive := range []string{"script-src", "connect-src"} {
		if source := cspDirective(csp, directive); !strings.Contains(source, "https://*.i.posthog.com") {
			t.Errorf("Content-Security-Policy %q directive = %q, want to contain PostHog hosts; csp=%q", directive, source, csp)
		}
	}
}

func TestMetricsRouteDisabledInSaaS(t *testing.T) {
	tests := []struct {
		name       string
		saas       bool
		wantStatus int
	}{
		{
			name:       "self-host exposes metrics",
			saas:       false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "SaaS does not expose metrics",
			saas:       true,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			if tt.saas {
				registerMetricsRoute(e, &config.Profile{SaaS: true}, nil, nil)
			} else {
				// Self-host: use a real store and license service so the
				// product-level license gauges are registered and collected.
				_, st, licenseService := newMetricsTestEcho(t)
				registerMetricsRoute(e, &config.Profile{SaaS: false}, st, licenseService)
			}

			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("GET /metrics status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func cspDirective(csp string, directive string) string {
	for _, source := range strings.Split(csp, ";") {
		source = strings.TrimSpace(source)
		if strings.HasPrefix(source, directive+" ") {
			return source
		}
	}
	return ""
}

func testSecurityHeadersCSP(t *testing.T, saas bool) string {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := securityHeadersMiddleware(saas)(func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)
	if err != nil {
		t.Fatal(err)
	}
	return rec.Header().Get("Content-Security-Policy")
}
