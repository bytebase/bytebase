package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v5"
)

func newFrontendTestFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":         &fstest.MapFile{Data: []byte("<!doctype html><html>APP</html>")},
		"assets/app-1234.js": &fstest.MapFile{Data: []byte("console.log('hi')")},
	}
}

func TestRegisterFrontendRoutes_AssetCacheHeaders(t *testing.T) {
	const wantImmutable = "public, max-age=31536000, immutable"
	cases := []struct {
		name             string
		path             string
		wantStatus       int
		wantCacheCtrl    string // "" = no Cache-Control header; "-" = don't assert
		wantBodyContains string
		bodyMustNotBe    string
	}{
		{
			name:          "real asset gets immutable cache",
			path:          "/assets/app-1234.js",
			wantStatus:    http.StatusOK,
			wantCacheCtrl: wantImmutable,
		},
		{
			// Cache-Control is intentionally unconstrained on 404: the header is
			// set before the handler runs, so a missing asset carries the immutable
			// directive too. That's fine — caching a 404 for an old asset hash is
			// harmless because new deploys reference new hashes.
			name:          "missing asset returns 404 not index.html",
			path:          "/assets/this-does-not-exist.js",
			wantStatus:    http.StatusNotFound,
			wantCacheCtrl: "-",
			bodyMustNotBe: "<!doctype html",
		},
		{
			name:             "spa route falls back to index.html with no immutable",
			path:             "/projects/foo",
			wantStatus:       http.StatusOK,
			wantCacheCtrl:    "",
			wantBodyContains: "APP",
		},
		{
			name:             "root serves index.html",
			path:             "/",
			wantStatus:       http.StatusOK,
			wantCacheCtrl:    "",
			wantBodyContains: "APP",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			registerFrontendRoutes(e, newFrontendTestFS())

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantCacheCtrl != "-" {
				if got := rec.Header().Get(echo.HeaderCacheControl); got != tc.wantCacheCtrl {
					t.Errorf("Cache-Control = %q, want %q", got, tc.wantCacheCtrl)
				}
			}
			body := rec.Body.String()
			if tc.wantBodyContains != "" && !strings.Contains(body, tc.wantBodyContains) {
				t.Errorf("body = %q, want to contain %q", body, tc.wantBodyContains)
			}
			if tc.bodyMustNotBe != "" && strings.Contains(strings.ToLower(body), tc.bodyMustNotBe) {
				t.Errorf("body should not contain %q (HTML5 fallback regression): %q", tc.bodyMustNotBe, body)
			}
		})
	}
}

func TestRegisterFrontendRoutes_AssetHeadRequest(t *testing.T) {
	e := echo.New()
	registerFrontendRoutes(e, newFrontendTestFS())

	req := httptest.NewRequest(http.MethodHead, "/assets/app-1234.js", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD /assets/app-1234.js status = %d, want 200; body=%q", rec.Code, rec.Body.String())
	}
	if got, want := rec.Header().Get(echo.HeaderCacheControl), "public, max-age=31536000, immutable"; got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

func TestRegisterFrontendRoutes_AssetGzip(t *testing.T) {
	body := strings.Repeat("console.log('hello gzip');\n", 100)
	distFS := fstest.MapFS{
		"index.html":         &fstest.MapFile{Data: []byte("<!doctype html>")},
		"assets/big-1234.js": &fstest.MapFile{Data: []byte(body)},
	}
	e := echo.New()
	registerFrontendRoutes(e, distFS)

	req := httptest.NewRequest(http.MethodGet, "/assets/big-1234.js", nil)
	req.Header.Set(echo.HeaderAcceptEncoding, "gzip")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get(echo.HeaderContentEncoding); got != "gzip" {
		t.Errorf("Content-Encoding = %q, want gzip", got)
	}
}
