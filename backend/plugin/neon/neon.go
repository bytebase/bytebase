// Package neon provides read-only discovery for one-time Neon imports.
package neon

import (
	"context"

	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const baseURL = "https://console.neon.tech/api/v2"
const maxPages = 100

var resourceID = regexp.MustCompile(`^[a-z0-9-]{1,60}$`)

// APIError contains only a status code; provider response bodies can contain secrets.
type APIError struct {
	StatusCode int
	RetryAfter time.Duration
}

var (
	ErrInvalidArgument     = errors.New("invalid Neon argument")
	ErrDatabaseNotFound    = errors.New("selected database does not exist on this Neon branch")
	ErrRoleNotFound        = errors.New("selected role does not exist on this Neon branch")
	ErrInvalidPageToken    = errors.New("invalid Neon page token")
	ErrEndpointUnavailable = errors.New("this Neon branch has no available read-write compute")
	ErrPasswordRequired    = errors.New("neon cannot reveal this role password; enter it manually")
	ErrUnavailable         = errors.New("could not reach Neon API")
	ErrInvalidResponse     = errors.New("unexpected Neon API response")
)

func (e *APIError) Error() string {
	switch e.StatusCode {
	case 401:
		return "Neon API key is invalid or expired"
	case 403:
		return "Neon API key does not permit this operation"
	case 404:
		return "Neon resource was not found"
	case 412:
		return "Neon cannot reveal this role's password; enter the database password manually"
	case 429:
		return "Neon API rate limit reached; try again later"
	default:
		return fmt.Sprintf("Neon API request failed (HTTP %d)", e.StatusCode)
	}
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Branch struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Database struct {
	Name      string `json:"name"`
	OwnerName string `json:"owner_name"`
}
type Role struct {
	Name string `json:"name"`
}
type Endpoint struct {
	ID       string `json:"id"`
	BranchID string `json:"branch_id"`
	Host     string `json:"host"`
	Type     string `json:"type"`
	Disabled bool   `json:"disabled"`
}
type DiscoveredBranch struct {
	ProjectID, ProjectName, ID, Name string
	Databases                        []Database
	Roles                            []Role
	EndpointAvailable                bool
	Error                            error
}
type Connection struct{ Host, Port, Database, Username, Password string }

// Client keeps an API key only for the lifetime of the requesting setup operation.
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey, baseURL: baseURL, httpClient: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (c *Client) get(ctx context.Context, path string, query url.Values, target any) error {
	if strings.TrimSpace(c.apiKey) == "" || strings.ContainsAny(c.apiKey, "\r\n") {
		return errors.Wrap(ErrInvalidArgument, "API key is required and must not contain line breaks")
	}
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return errors.New("could not construct Neon request")
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var timeout net.Error
		if errors.As(err, &timeout) && timeout.Timeout() {
			return context.DeadlineExceeded
		}
		return ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		retryAfter := time.Duration(0)
		if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 {
			retryAfter = time.Duration(seconds) * time.Second
		} else if deadline, err := http.ParseTime(resp.Header.Get("Retry-After")); err == nil {
			retryAfter = max(0, time.Until(deadline))
		}
		return &APIError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024+1))
	if err != nil {
		return ErrUnavailable
	}
	if len(body) > 4*1024*1024 {
		return errors.Wrap(ErrInvalidResponse, "could not read Neon API response")
	}
	if err := json.Unmarshal(body, target); err != nil {
		return errors.Wrap(ErrInvalidResponse, "invalid Neon API response")
	}
	return nil
}

// ValidResourceID reports whether id has the shape of a Neon project or branch ID.
func ValidResourceID(id string) bool {
	return resourceID.MatchString(id)
}

func validIDs(ids ...string) error {
	for _, id := range ids {
		if !ValidResourceID(id) {
			return errors.Wrap(ErrInvalidArgument, "invalid Neon resource ID")
		}
	}
	return nil
}
func branchPath(projectID, branchID string) (string, error) {
	if err := validIDs(projectID, branchID); err != nil {
		return "", err
	}
	return "/projects/" + projectID + "/branches/" + branchID, nil
}
func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var response struct {
		Organizations []Organization `json:"organizations"`
	}
	if err := c.get(ctx, "/users/me/organizations", nil, &response); err != nil {
		return nil, err
	}
	if response.Organizations == nil {
		return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon organizations response")
	}
	for _, organization := range response.Organizations {
		if err := validIDs(organization.ID); err != nil {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon organization")
		}
	}
	return response.Organizations, nil
}
func (c *Client) ListProjects(ctx context.Context, orgID string) ([]Project, error) {
	query := url.Values{"limit": {"100"}}
	if orgID != "" {
		if err := validIDs(orgID); err != nil {
			return nil, err
		}
		query.Set("org_id", orgID)
	}
	var result []Project
	seen := map[string]bool{}
	for range maxPages {
		var response struct {
			Projects    []Project `json:"projects"`
			Unavailable []string  `json:"unavailable_project_ids"`
			Pagination  struct {
				Cursor string `json:"cursor"`
			} `json:"pagination"`
		}
		if err := c.get(ctx, "/projects", query, &response); err != nil {
			return nil, err
		}
		if response.Projects == nil {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon projects response")
		}
		if len(response.Unavailable) > 0 {
			return nil, errors.New("some Neon projects are temporarily unavailable; try discovery again")
		}
		for _, project := range response.Projects {
			if validIDs(project.ID) != nil || project.Name == "" {
				return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon project")
			}
		}
		result = append(result, response.Projects...)
		cursor := response.Pagination.Cursor
		if len(response.Projects) == 0 || cursor == "" {
			return result, nil
		}
		if seen[cursor] {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon pagination cursor")
		}
		seen[cursor] = true
		query.Set("cursor", cursor)
	}
	return nil, errors.New("neon project discovery exceeded its page limit")
}
func (c *Client) ListBranches(ctx context.Context, projectID string) ([]Branch, error) {
	if err := validIDs(projectID); err != nil {
		return nil, err
	}
	query := url.Values{"limit": {"100"}, "sort_by": {"created_at"}, "sort_order": {"asc"}}
	var result []Branch
	seen := map[string]bool{}
	for range maxPages {
		var response struct {
			Branches   []Branch `json:"branches"`
			Pagination struct {
				Next string `json:"next"`
			} `json:"pagination"`
		}
		if err := c.get(ctx, "/projects/"+projectID+"/branches", query, &response); err != nil {
			return nil, err
		}
		if response.Branches == nil {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon branches response")
		}
		for _, branch := range response.Branches {
			if validIDs(branch.ID) != nil || branch.Name == "" {
				return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon branch")
			}
		}
		result = append(result, response.Branches...)
		cursor := response.Pagination.Next
		if cursor == "" {
			return result, nil
		}
		if seen[cursor] {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon pagination cursor")
		}
		seen[cursor] = true
		query.Set("cursor", cursor)
	}
	return nil, errors.New("neon branch discovery exceeded its page limit")
}
func (c *Client) ListDatabases(ctx context.Context, projectID, branchID string) ([]Database, error) {
	path, err := branchPath(projectID, branchID)
	if err != nil {
		return nil, err
	}
	var response struct {
		Databases []Database `json:"databases"`
	}
	if err := c.get(ctx, path+"/databases", nil, &response); err != nil {
		return nil, err
	}
	if response.Databases == nil {
		return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon databases response")
	}
	for _, db := range response.Databases {
		if db.Name == "" || db.OwnerName == "" {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon database")
		}
	}
	return response.Databases, nil
}
func (c *Client) ListRoles(ctx context.Context, projectID, branchID string) ([]Role, error) {
	path, err := branchPath(projectID, branchID)
	if err != nil {
		return nil, err
	}
	var response struct {
		Roles []Role `json:"roles"`
	}
	if err := c.get(ctx, path+"/roles", nil, &response); err != nil {
		return nil, err
	}
	if response.Roles == nil {
		return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon roles response")
	}
	for _, role := range response.Roles {
		if role.Name == "" {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon role")
		}
	}
	return response.Roles, nil
}
func (c *Client) ListEndpoints(ctx context.Context, projectID, branchID string) ([]Endpoint, error) {
	path, err := branchPath(projectID, branchID)
	if err != nil {
		return nil, err
	}
	var response struct {
		Endpoints []Endpoint `json:"endpoints"`
	}
	if err := c.get(ctx, path+"/endpoints", nil, &response); err != nil {
		return nil, err
	}
	if response.Endpoints == nil {
		return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon endpoints response")
	}
	return response.Endpoints, nil
}
func (c *Client) GetRolePassword(ctx context.Context, projectID, branchID, roleName string) (string, error) {
	path, err := branchPath(projectID, branchID)
	if err != nil {
		return "", err
	}
	if roleName == "" {
		return "", errors.Wrap(ErrInvalidArgument, "role is required")
	}
	var response struct {
		Password string `json:"password"`
	}
	if err := c.get(ctx, path+"/roles/"+url.PathEscape(roleName)+"/reveal_password", nil, &response); err != nil {
		return "", err
	}
	if response.Password == "" {
		return "", ErrPasswordRequired
	}
	return response.Password, nil
}

func directEndpoint(endpoints []Endpoint, branchID string) (*Endpoint, error) {
	for _, endpoint := range endpoints {
		if endpoint.Type != "read_write" || endpoint.Disabled || endpoint.BranchID != branchID {
			continue
		}
		host := endpoint.Host
		if !strings.HasSuffix(host, ".neon.tech") || strings.ContainsAny(host, "/:@?#\\ \t\r\n") || strings.Contains(strings.Split(host, ".")[0], "-pooler") {
			return nil, errors.Wrap(ErrEndpointUnavailable, "invalid direct endpoint")
		}
		return &endpoint, nil
	}
	return nil, ErrEndpointUnavailable
}

// Resolve verifies the selected role/database and retrieves credentials without modifying Neon.
func (c *Client) Resolve(ctx context.Context, projectID, branchID, roleName, databaseName string, manualPassword *string) (*Connection, error) {
	databases, err := c.ListDatabases(ctx, projectID, branchID)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(databases, func(db Database) bool { return db.Name == databaseName }) {
		return nil, ErrDatabaseNotFound
	}
	roles, err := c.ListRoles(ctx, projectID, branchID)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(roles, func(role Role) bool { return role.Name == roleName }) {
		return nil, ErrRoleNotFound
	}
	endpoints, err := c.ListEndpoints(ctx, projectID, branchID)
	if err != nil {
		return nil, err
	}
	endpoint, err := directEndpoint(endpoints, branchID)
	if err != nil {
		return nil, err
	}
	var password string
	if manualPassword != nil {
		password = *manualPassword
	} else {
		password, err = c.GetRolePassword(ctx, projectID, branchID, roleName)
		if err != nil {
			return nil, err
		}
	}
	if password == "" {
		return nil, ErrPasswordRequired
	}
	return &Connection{Host: endpoint.Host, Port: "5432", Database: databaseName, Username: roleName, Password: password}, nil
}
