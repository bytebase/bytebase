package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// ClientStore stores dynamically registered OAuth clients.
type ClientStore struct {
	mu      sync.RWMutex
	clients map[string]*RegisteredClient
}

// RegisteredClient represents a dynamically registered OAuth client.
type RegisteredClient struct {
	ClientID                string    `json:"client_id"`
	ClientSecret            string    `json:"client_secret,omitempty"`
	ClientName              string    `json:"client_name,omitempty"`
	RedirectURIs            []string  `json:"redirect_uris"`
	GrantTypes              []string  `json:"grant_types,omitempty"`
	ResponseTypes           []string  `json:"response_types,omitempty"`
	TokenEndpointAuthMethod string    `json:"token_endpoint_auth_method,omitempty"`
	ClientIDIssuedAt        int64     `json:"client_id_issued_at,omitempty"`
	RegisteredAt            time.Time `json:"-"`
}

// NewClientStore creates a new client store.
func NewClientStore() *ClientStore {
	return &ClientStore{
		clients: make(map[string]*RegisteredClient),
	}
}

// Get retrieves a client by ID.
func (s *ClientStore) Get(clientID string) (*RegisteredClient, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[clientID]
	return client, ok
}

// Register stores a new client.
func (s *ClientStore) Register(client *RegisteredClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client.ClientID] = client
}

// RegistrationRequest is the dynamic client registration request body.
type RegistrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

// RegisterHandler handles dynamic client registration (RFC 7591).
type RegisterHandler struct {
	clientStore *ClientStore
}

// NewRegisterHandler creates a new registration handler.
func NewRegisterHandler(clientStore *ClientStore) *RegisterHandler {
	return &RegisterHandler{clientStore: clientStore}
}

// ServeHTTP handles POST /oauth/register
func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRegistrationError(w, "invalid_client_metadata", "Invalid request body")
		return
	}

	// Validate redirect_uris - required per RFC 7591
	if len(req.RedirectURIs) == 0 {
		writeRegistrationError(w, "invalid_redirect_uri", "redirect_uris is required")
		return
	}

	// Generate client_id
	clientID, err := generateRandomString(16)
	if err != nil {
		writeRegistrationError(w, "server_error", "Failed to generate client_id")
		return
	}

	// Set defaults
	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}

	authMethod := req.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "none"
	}

	now := time.Now()
	client := &RegisteredClient{
		ClientID:                clientID,
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: authMethod,
		ClientIDIssuedAt:        now.Unix(),
		RegisteredAt:            now,
	}

	h.clientStore.Register(client)

	// Return registration response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(client)
}

func writeRegistrationError(w http.ResponseWriter, errorCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             errorCode,
		"error_description": description,
	})
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
