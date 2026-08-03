package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestReauthorizeRejectsCurrentAccessToken(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-test');
		INSERT INTO principal (name, email, password_hash) VALUES ('demo', 'test@example.com', 'unused');
		INSERT INTO oauth2_client (client_id, workspace, client_secret_hash, config)
		VALUES ('client-A', NULL, 'unused-hash', '{"clientName":"test","redirectUris":["http://localhost/cb"],"grantTypes":["authorization_code","refresh_token"],"tokenEndpointAuthMethod":"none"}'::jsonb);
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	st, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	refreshToken := "refresh-token"
	_, err = st.CreateOAuth2RefreshToken(ctx, &store.OAuth2RefreshTokenMessage{
		TokenHash: auth.HashToken(refreshToken),
		ClientID:  "client-A",
		UserEmail: "test@example.com",
		Workspace: "ws-test",
		Config:    &storepb.OAuth2RefreshTokenConfig{},
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	const secret = "test-secret-key"
	s, err := NewServer(st, &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}, secret)
	require.NoError(t, err)

	accessToken := generateOAuth2MCPToken(t, secret, "client-A", "ws-test")
	reauthorizeCtx := withAccessToken(ctx, accessToken)
	reauthorizeCtx = withUserEmail(reauthorizeCtx, "test@example.com")
	reauthorizeCtx = withOAuth2ClientID(reauthorizeCtx, "client-A")
	reauthorizeCtx = withWorkspaceID(reauthorizeCtx, "ws-test")
	_, _, err = s.handleReauthorize(reauthorizeCtx, nil, ReauthorizeInput{})
	require.NoError(t, err)

	stored, err := st.GetOAuth2RefreshToken(ctx, "client-A", auth.HashToken(refreshToken))
	require.NoError(t, err)
	require.Nil(t, stored)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := s.authMiddleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "success")
	})
	if err := handler(c); err != nil {
		echo.DefaultHTTPErrorHandler(true)(c, err)
	}

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Header().Get("WWW-Authenticate"), `error="invalid_token"`)
}
