package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
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
			} {
				if got := strings.Contains(csp, googleSource); got != tt.wantGoogleGA4 {
					t.Errorf("Content-Security-Policy contains %q = %t, want %t; csp=%q", googleSource, got, tt.wantGoogleGA4, csp)
				}
			}
		})
	}
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
