// Package oauth provides OAuth 2.1 authorization server for MCP.
package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// MetadataServer serves OAuth metadata endpoints.
type MetadataServer struct {
	issuer string
}

// NewMetadataServer creates a new metadata server.
func NewMetadataServer(issuer string) *MetadataServer {
	return &MetadataServer{issuer: issuer}
}

// ProtectedResourceMetadata handles GET /.well-known/oauth-protected-resource
func (s *MetadataServer) ProtectedResourceMetadata(w http.ResponseWriter, _ *http.Request) {
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               s.issuer + "/mcp",
		AuthorizationServers:   []string{s.issuer},
		ScopesSupported:        []string{"mcp:tools"},
		BearerMethodsSupported: []string{"header"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_ = json.NewEncoder(w).Encode(metadata)
}

// AuthorizationServerMetadata handles GET /.well-known/oauth-authorization-server
func (s *MetadataServer) AuthorizationServerMetadata(w http.ResponseWriter, _ *http.Request) {
	metadata := map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/oauth/authorize",
		"token_endpoint":                        s.issuer + "/oauth/token",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{"mcp:tools"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_ = json.NewEncoder(w).Encode(metadata)
}
