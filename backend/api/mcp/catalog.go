package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

// catalogPath serves the tool list and routing instructions without a token,
// for clients that must list tools before signing in. It is the only MCP path
// that answers without a token (docs/design/mcp-public-catalog.md).
const catalogPath = mcpResourcePath + "/catalog"

type catalogDocument struct {
	Instructions            string      `json:"instructions"`
	InstructionsWithoutForm string      `json:"instructionsWithoutForm"`
	Tools                   []*mcp.Tool `json:"tools"`
}

// buildCatalog lists the registered tools over an in-memory session, so the
// catalog is exactly what a session's tools/list returns. It reads nothing but
// the registry and two constants: the response must never carry anything an
// instance knows about itself.
func buildCatalog(server *mcp.Server) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open the catalog's server session")
	}
	defer serverSession.Close()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "bytebase-catalog"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open the catalog's client session")
	}
	defer session.Close()

	tools := []*mcp.Tool{}
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return nil, errors.Wrap(err, "failed to list tools for the catalog")
		}
		tools = append(tools, tool)
	}
	return json.Marshal(catalogDocument{
		Instructions:            serverInstructions,
		InstructionsWithoutForm: askInChatInstructions,
		Tools:                   tools,
	})
}

func (s *Server) serveCatalog(c *echo.Context) error {
	c.Response().Header().Set("Cache-Control", "public, max-age=3600")
	return c.Blob(http.StatusOK, echo.MIMEApplicationJSON, s.catalog)
}
