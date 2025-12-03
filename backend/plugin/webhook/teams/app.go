// This file implements Teams direct messaging using Microsoft Graph API and Bot Framework.
//
// The flow for sending proactive messages to Teams users:
// 1. Look up user by email using Graph API (User.Read.All permission)
// 2. Get Teams app ID from organization's app catalog (AppCatalog.Read.All permission)
// 3. Install app for user if not already installed (TeamsAppInstallation.ReadWriteSelfForUser.All permission)
// 4. Get chat ID from the app installation (TeamsAppInstallation.ReadWriteSelfForUser.All permission)
// 5. Send message via Bot Framework API using the chat ID
//
// Documentation:
// - Microsoft Graph API: https://learn.microsoft.com/en-us/graph/overview
// - Graph API User Lookup: https://learn.microsoft.com/en-us/graph/api/user-get
// - Graph API App Installation: https://learn.microsoft.com/en-us/graph/api/userteamwork-post-installedapps
// - Graph API Get Chat: https://learn.microsoft.com/en-us/graph/api/userscopeteamsappinstallation-get-chat
// - Bot Framework REST API: https://learn.microsoft.com/en-us/azure/bot-service/rest-api/bot-framework-rest-connector-send-and-receive-messages
// - Azure Bot Service: https://learn.microsoft.com/en-us/azure/bot-service/
// - OAuth2 Client Credentials Flow: https://learn.microsoft.com/en-us/entra/identity-platform/v2-oauth2-client-creds-grant-flow
package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
)

type provider struct {
	tenantID     string
	clientID     string
	clientSecret string
	c            *http.Client
	graphToken   string
	botToken     string
}

func newProvider(tenantID, clientID, clientSecret string) *provider {
	return &provider{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
		c:            &http.Client{},
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type tokenKey struct {
	tenantID     string
	clientID     string
	clientSecret string
	scope        string
}

type tokenValue struct {
	token    string
	expireAt time.Time
}

var (
	tokenCacheLock sync.Mutex
	tokenCache     = func() *lru.Cache[tokenKey, *tokenValue] {
		cache, err := lru.New[tokenKey, *tokenValue](10)
		if err != nil {
			panic(err)
		}
		return cache
	}()
)

const (
	graphScope = "https://graph.microsoft.com/.default"
	botScope   = "https://api.botframework.com/.default"
)

// getToken fetches an OAuth2 token using client credentials flow.
func getToken(ctx context.Context, c *http.Client, tenantID, clientID, clientSecret, scope string) (*tokenValue, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("scope", scope)
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to construct token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to request token")
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read token response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("token request failed (status %d): %s", resp.StatusCode, string(b))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(b, &tokenResp); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal token response")
	}

	return &tokenValue{
		token:    tokenResp.AccessToken,
		expireAt: time.Now().Add(time.Second * time.Duration(tokenResp.ExpiresIn)),
	}, nil
}

func getTokenCached(ctx context.Context, c *http.Client, tenantID, clientID, clientSecret, scope string) (string, error) {
	tokenCacheLock.Lock()
	defer tokenCacheLock.Unlock()

	key := tokenKey{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
		scope:        scope,
	}

	token, ok := tokenCache.Get(key)
	if ok && time.Now().Before(token.expireAt.Add(-time.Minute)) {
		return token.token, nil
	}

	token, err := getToken(ctx, c, tenantID, clientID, clientSecret, scope)
	if err != nil {
		return "", err
	}
	tokenCache.Add(key, token)

	return token.token, nil
}

func (p *provider) refreshGraphToken(ctx context.Context) error {
	token, err := getTokenCached(ctx, p.c, p.tenantID, p.clientID, p.clientSecret, graphScope)
	if err != nil {
		return err
	}
	p.graphToken = token
	return nil
}

func (p *provider) refreshBotToken(ctx context.Context) error {
	token, err := getTokenCached(ctx, p.c, p.tenantID, p.clientID, p.clientSecret, botScope)
	if err != nil {
		return err
	}
	p.botToken = token
	return nil
}

// userResponse is the response from Microsoft Graph API for user lookup.
type userResponse struct {
	ID                string `json:"id"`
	UserPrincipalName string `json:"userPrincipalName"`
	Mail              string `json:"mail"`
}

// teamsAppInstallation represents an installed Teams app.
type teamsAppInstallation struct {
	ID string `json:"id"`
}

// teamsAppInstallationsResponse is the response for listing installed apps.
type teamsAppInstallationsResponse struct {
	Value []teamsAppInstallation `json:"value"`
}

// chatResponse is the response from the chat endpoint.
type chatResponse struct {
	ID string `json:"id"`
}

var userIDCache = func() *lru.Cache[string, string] {
	cache, err := lru.New[string, string](5000)
	if err != nil {
		panic(err)
	}
	return cache
}()

// getIDByEmail gets user AAD object ID by email using Microsoft Graph API.
// https://learn.microsoft.com/en-us/graph/api/user-get
func (p *provider) getIDByEmail(ctx context.Context, emails []string) (map[string]string, error) {
	userID := make(map[string]string)
	var emailsToGet []string

	for _, email := range emails {
		id, ok := userIDCache.Get(email)
		if ok {
			if id != "" {
				userID[email] = id
			}
		} else {
			emailsToGet = append(emailsToGet, email)
		}
	}

	if len(emailsToGet) == 0 {
		return userID, nil
	}

	if p.graphToken == "" {
		if err := p.refreshGraphToken(ctx); err != nil {
			return nil, errors.Wrapf(err, "failed to refresh graph token")
		}
	}

	for _, email := range emailsToGet {
		id, err := p.getUserIDByEmail(ctx, email)
		if err != nil {
			// Cache empty string for not found users to avoid repeated lookups.
			userIDCache.Add(email, "")
			continue
		}
		userID[email] = id
		userIDCache.Add(email, id)
	}

	return userID, nil
}

// usersResponse is the response from Microsoft Graph API for user list queries.
type usersResponse struct {
	Value []userResponse `json:"value"`
}

func (p *provider) getUserIDByEmail(ctx context.Context, email string) (string, error) {
	// First, try to get user directly by userPrincipalName.
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s", url.PathEscape(email))

	b, err := p.doGraphRequest(ctx, http.MethodGet, apiURL, nil)
	if err == nil {
		var user userResponse
		if err := json.Unmarshal(b, &user); err == nil && user.ID != "" {
			return user.ID, nil
		}
	}

	// If direct lookup fails, search by mail field (needed for guest users).
	// Guest users have UPNs like "user_domain.com#EXT#@tenant.onmicrosoft.com"
	// but their mail field contains the original email.
	filterURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users?$filter=mail%%20eq%%20'%s'", url.QueryEscape(email))
	b, err = p.doGraphRequest(ctx, http.MethodGet, filterURL, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to search user by mail")
	}

	var users usersResponse
	if err := json.Unmarshal(b, &users); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal users response")
	}

	if len(users.Value) == 0 {
		return "", errors.Errorf("user with email %s not found", email)
	}

	return users.Value[0].ID, nil
}

// getTeamsAppID retrieves the Teams app ID from the app catalog.
// The app must be published to the organization's app catalog.
func (p *provider) getTeamsAppID(ctx context.Context) (string, error) {
	// The externalId in the app catalog is the same as the clientID (Azure AD app ID).
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/appCatalogs/teamsApps?$filter=externalId%%20eq%%20'%s'", p.clientID)

	b, err := p.doGraphRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to query app catalog")
	}

	var resp struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal app catalog response")
	}

	if len(resp.Value) == 0 {
		return "", errors.Errorf("Teams app with externalId %s not found in app catalog", p.clientID)
	}

	return resp.Value[0].ID, nil
}

// ensureAppInstalledForUser ensures the Teams app is installed for a user.
// Returns the installation ID.
func (p *provider) ensureAppInstalledForUser(ctx context.Context, userAADID, teamsAppID string) (string, error) {
	// First, check if the app is already installed.
	listURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/teamwork/installedApps?$filter=teamsApp/externalId%%20eq%%20'%s'&$expand=teamsApp", userAADID, p.clientID)

	b, err := p.doGraphRequest(ctx, http.MethodGet, listURL, nil)
	if err == nil {
		var resp teamsAppInstallationsResponse
		if err := json.Unmarshal(b, &resp); err == nil && len(resp.Value) > 0 {
			return resp.Value[0].ID, nil
		}
	}

	// App not installed, install it.
	installURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/teamwork/installedApps", userAADID)
	installBody := fmt.Sprintf(`{"teamsApp@odata.bind":"https://graph.microsoft.com/v1.0/appCatalogs/teamsApps/%s"}`, teamsAppID)

	_, err = p.doGraphRequest(ctx, http.MethodPost, installURL, []byte(installBody))
	if err != nil {
		return "", errors.Wrapf(err, "failed to install Teams app for user")
	}

	// Retrieve the installation ID.
	b, err = p.doGraphRequest(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get installation after install")
	}

	var resp teamsAppInstallationsResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal installation response")
	}

	if len(resp.Value) == 0 {
		return "", errors.Errorf("installation not found after installing app")
	}

	return resp.Value[0].ID, nil
}

// getChatIDForUser gets the chat ID for a user's app installation.
func (p *provider) getChatIDForUser(ctx context.Context, userAADID, installationID string) (string, error) {
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/teamwork/installedApps/%s/chat", userAADID, installationID)

	b, err := p.doGraphRequest(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get chat for installation")
	}

	var resp chatResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal chat response")
	}

	return resp.ID, nil
}

func (p *provider) doGraphRequest(ctx context.Context, method, apiURL string, data []byte) ([]byte, error) {
	const maxRetries = 3

	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		b, retry, err := func() ([]byte, bool, error) {
			var body io.Reader
			if data != nil {
				body = bytes.NewReader(data)
			}

			req, err := http.NewRequestWithContext(ctx, method, apiURL, body)
			if err != nil {
				return nil, false, errors.Wrapf(err, "failed to construct request")
			}

			req.Header.Set("Authorization", "Bearer "+p.graphToken)
			if data != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := p.c.Do(req)
			if err != nil {
				return nil, false, errors.Wrapf(err, "request failed")
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, false, errors.Wrapf(err, "failed to read response")
			}

			if resp.StatusCode == http.StatusUnauthorized {
				if err := p.refreshGraphToken(ctx); err != nil {
					return nil, false, errors.Wrapf(err, "failed to refresh token")
				}
				return nil, true, nil
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return nil, false, errors.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
			}

			return respBody, false, nil
		}()

		if err != nil {
			return nil, err
		}
		if retry {
			continue
		}
		return b, nil
	}

	return nil, errors.Errorf("exceeded max retries for %s %s", method, apiURL)
}

// Bot Framework types for messaging.

type activity struct {
	Type        string       `json:"type"`
	Text        string       `json:"text,omitempty"`
	Attachments []attachment `json:"attachments,omitempty"`
}

type attachment struct {
	ContentType string `json:"contentType"`
	Content     any    `json:"content"`
}

// AdaptiveCard represents a Microsoft Adaptive Card.
// Adaptive Card schema: https://adaptivecards.io/explorer/
// Adaptive Card designer: https://adaptivecards.io/designer/
// Adaptive Card documentation: https://learn.microsoft.com/en-us/adaptive-cards/
type AdaptiveCard struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	Body    []any  `json:"body"`
	Actions []any  `json:"actions,omitempty"`
}

type textBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Size   string `json:"size,omitempty"`
	Weight string `json:"weight,omitempty"`
	Wrap   bool   `json:"wrap,omitempty"`
}

type factSet struct {
	Type  string `json:"type"`
	Facts []fact `json:"facts"`
}

type fact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type actionOpenURL struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

const (
	// serviceURL is the Bot Framework service URL for Teams.
	// https://learn.microsoft.com/en-us/microsoftteams/platform/bots/how-to/conversations/send-proactive-messages#get-the-user-id-team-id-or-channel-id
	serviceURL = "https://smba.trafficmanager.net/teams"
)

// sendMessage sends a message to a user via Bot Framework.
// It ensures the Teams app is installed for the user first, then sends the message
// using the chat ID obtained from the installation.
func (p *provider) sendMessage(ctx context.Context, userAADID string, card *AdaptiveCard) error {
	// First, ensure we have a Graph token.
	if p.graphToken == "" {
		if err := p.refreshGraphToken(ctx); err != nil {
			return errors.Wrapf(err, "failed to refresh graph token")
		}
	}

	// Get the Teams app ID from the app catalog.
	teamsAppID, err := p.getTeamsAppID(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get Teams app ID from catalog")
	}

	// Ensure the app is installed for the user.
	installationID, err := p.ensureAppInstalledForUser(ctx, userAADID, teamsAppID)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure app is installed for user")
	}

	// Get the chat ID for the user's app installation.
	chatID, err := p.getChatIDForUser(ctx, userAADID, installationID)
	if err != nil {
		return errors.Wrapf(err, "failed to get chat ID for user")
	}

	// Send the message via Bot Framework using the chat ID as conversation ID.
	return p.sendMessageToConversation(ctx, chatID, card)
}

// sendMessageToConversation sends an Adaptive Card message to a conversation via Bot Framework.
func (p *provider) sendMessageToConversation(ctx context.Context, conversationID string, card *AdaptiveCard) error {
	if p.botToken == "" {
		if err := p.refreshBotToken(ctx); err != nil {
			return errors.Wrapf(err, "failed to refresh bot token")
		}
	}

	apiURL := fmt.Sprintf("%s/v3/conversations/%s/activities", serviceURL, conversationID)

	act := activity{
		Type: "message",
		Attachments: []attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content:     card,
			},
		},
	}

	data, err := json.Marshal(act)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal activity")
	}

	_, err = p.doBotRequest(ctx, http.MethodPost, apiURL, data)
	if err != nil {
		return errors.Wrapf(err, "failed to send message to conversation")
	}

	return nil
}

func (p *provider) doBotRequest(ctx context.Context, method, apiURL string, data []byte) ([]byte, error) {
	const maxRetries = 3

	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		b, retry, err := func() ([]byte, bool, error) {
			req, err := http.NewRequestWithContext(ctx, method, apiURL, bytes.NewReader(data))
			if err != nil {
				return nil, false, errors.Wrapf(err, "failed to construct request")
			}

			req.Header.Set("Authorization", "Bearer "+p.botToken)
			req.Header.Set("Content-Type", "application/json")

			resp, err := p.c.Do(req)
			if err != nil {
				return nil, false, errors.Wrapf(err, "request failed")
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, false, errors.Wrapf(err, "failed to read response")
			}

			if resp.StatusCode == http.StatusUnauthorized {
				if err := p.refreshBotToken(ctx); err != nil {
					return nil, false, errors.Wrapf(err, "failed to refresh token")
				}
				return nil, true, nil
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return nil, false, errors.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
			}

			return respBody, false, nil
		}()

		if err != nil {
			return nil, err
		}
		if retry {
			continue
		}
		return b, nil
	}

	return nil, errors.Errorf("exceeded max retries for %s %s", method, apiURL)
}

// Validate validates the Teams configuration by attempting to get a token and look up a user.
func Validate(ctx context.Context, tenantID, clientID, clientSecret, email string) error {
	// Trim whitespace from all inputs to prevent copy-paste issues.
	tenantID = strings.TrimSpace(tenantID)
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	email = strings.TrimSpace(email)

	p := newProvider(tenantID, clientID, clientSecret)

	if err := p.refreshGraphToken(ctx); err != nil {
		return errors.Wrapf(err, "failed to get Graph API token")
	}

	_, err := p.getIDByEmail(ctx, []string{email})
	if err != nil {
		return errors.Wrapf(err, "failed to look up user %s", email)
	}

	return nil
}
