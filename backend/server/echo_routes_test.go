package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestSecurityHeadersMiddleware_AllowsGA4CloudTag(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := securityHeadersMiddleware(func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)
	if err != nil {
		t.Fatal(err)
	}

	csp := rec.Header().Get("Content-Security-Policy")
	for _, want := range []string{
		"script-src 'self'",
		"https://www.googletagmanager.com",
		"connect-src 'self'",
		"https://*.google-analytics.com",
		"img-src 'self'",
		"https://*.google-analytics.com",
	} {
		if !strings.Contains(csp, want) {
			t.Errorf("Content-Security-Policy = %q, want to contain %q", csp, want)
		}
	}
}
