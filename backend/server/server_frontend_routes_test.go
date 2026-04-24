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
		wantCacheCtrl    string // "" = asserts no Cache-Control header
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
			// 404 must NOT carry the immutable Cache-Control header. In an HA
			// rolling deploy, an older replica can 404 a new hashed asset briefly;
			// browsers/CDNs caching that 404 for a year would wedge users on a
			// broken app long after the deploy completes.
			name:          "missing asset returns 404 with no immutable cache",
			path:          "/assets/this-does-not-exist.js",
			wantStatus:    http.StatusNotFound,
			wantCacheCtrl: "",
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
			if got := rec.Header().Get(echo.HeaderCacheControl); got != tc.wantCacheCtrl {
				t.Errorf("Cache-Control = %q, want %q", got, tc.wantCacheCtrl)
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

func TestDefaultAPIRequestSkipper(t *testing.T) {
	cases := map[string]bool{
		// API-ish prefixes that must be skipped (fall through to registered handlers).
		"/api/v1/login":                           true,
		"/v1/projects":                            true,
		"/v1:adminExecute":                        true,
		"/bytebase.v1.AuthService/Login":          true,
		"/bytebase.v1.ProjectService/GetProject":  true,
		"/.well-known/oauth-authorization-server": true,
		"/hook/gitlab":                            true,
		// Paths that must NOT be skipped (they're meant for the static handler).
		"/assets/main-abc.js": false,
		"/projects/foo":       false,
		"/":                   false,
	}
	for reqPath, want := range cases {
		t.Run(reqPath, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, reqPath, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if got := defaultAPIRequestSkipper(c); got != want {
				t.Errorf("defaultAPIRequestSkipper(%q) = %v, want %v", reqPath, got, want)
			}
		})
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
