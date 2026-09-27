package oauth2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
)

func TestRegisterRejectsConfidentialClient(t *testing.T) {
	s := &Service{profile: &config.Profile{ExternalURL: "https://bb.example.com"}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/oauth2/register", strings.NewReader(`{
		"client_name":"test client",
		"redirect_uris":["http://localhost/callback"],
		"grant_types":["authorization_code"],
		"token_endpoint_auth_method":"client_secret_basic"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, s.handleRegister(c))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "invalid_client_metadata", body["error"])
}

// TestRegisterBodyLimit verifies that oversized request bodies are rejected
// before JSON binding. Echo's default JSON deserializer buffers the whole
// body in memory and /register is unauthenticated, so the route-level
// BodyLimit middleware is the only thing bounding attacker-controlled
// allocation on this endpoint.
func TestRegisterBodyLimit(t *testing.T) {
	s := &Service{profile: &config.Profile{ExternalURL: "https://bb.example.com"}}
	e := echo.New()
	s.RegisterRoutes(e)

	oversized := `{"client_name":"` + strings.Repeat("a", maxOAuth2BodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth2/register", strings.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)

	// A body under the limit must pass the middleware and reach the handler.
	// Malformed JSON keeps the handler from touching the (nil) store while
	// still proving the request was bound rather than rejected on size.
	req = httptest.NewRequest(http.MethodPost, "/api/oauth2/register", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
