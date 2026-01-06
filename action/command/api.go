package command

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialInterval is the initial retry interval
	InitialInterval time.Duration
	// MaxInterval is the maximum retry interval
	MaxInterval time.Duration
}

// ClientOptions configures the Client behavior
type ClientOptions struct {
	// PageSize controls the number of items per page in list operations
	PageSize int32
	// HTTPClient allows providing a custom HTTP client
	HTTPClient *http.Client
	// Timeout for RPC calls (applies only if HTTPClient is not provided)
	Timeout time.Duration
	// RetryConfig configures retry behavior for transient errors
	RetryConfig *RetryConfig
}

// DefaultClientOptions returns the default client options
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		PageSize: 100,
		Timeout:  120 * time.Second,
		RetryConfig: &RetryConfig{
			MaxAttempts:     3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
		},
	}
}

// Client is the API message for Bytebase API Client.
type Client struct {
	// HTTP client for Connect RPC
	httpClient *http.Client

	// Base URL
	url string

	// Credentials for re-authentication
	serviceAccount       string
	serviceAccountSecret string

	// Connect RPC service clients
	releaseClient  v1connect.ReleaseServiceClient
	planClient     v1connect.PlanServiceClient
	rolloutClient  v1connect.RolloutServiceClient
	actuatorClient v1connect.ActuatorServiceClient

	// Client options
	options ClientOptions
}

// NewClient returns the new Bytebase API client with default options.
func NewClient(url, serviceAccount, serviceAccountSecret string) (*Client, error) {
	return NewClientWithOptions(url, serviceAccount, serviceAccountSecret, DefaultClientOptions())
}

// NewClientWithOptions returns a new Bytebase API client with custom options.
func NewClientWithOptions(url, serviceAccount, serviceAccountSecret string, opts ClientOptions) (*Client, error) {
	if opts.PageSize <= 0 {
		opts.PageSize = 100
	}
	if opts.PageSize > 1000 {
		opts.PageSize = 1000
	}

	// Use provided HTTP client or create a new one
	httpClient := opts.HTTPClient
	if httpClient == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 120 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	// Create token refresher function
	tokenRefresher := getTokenRefresher(httpClient, serviceAccount, serviceAccountSecret, url)

	// Create unified interceptor
	unifiedInt := newUnifiedInterceptor(opts.RetryConfig, tokenRefresher)
	interceptors := connect.WithInterceptors(unifiedInt)

	c := Client{
		httpClient:           httpClient,
		url:                  url,
		serviceAccount:       serviceAccount,
		serviceAccountSecret: serviceAccountSecret,
		options:              opts,
		releaseClient:        v1connect.NewReleaseServiceClient(httpClient, url, interceptors),
		planClient:           v1connect.NewPlanServiceClient(httpClient, url, interceptors),
		rolloutClient:        v1connect.NewRolloutServiceClient(httpClient, url, interceptors),
		actuatorClient:       v1connect.NewActuatorServiceClient(httpClient, url, interceptors),
	}

	return &c, nil
}

func getTokenRefresher(httpClient connect.HTTPClient, email, password, url string) func(ctx context.Context) (string, error) {
	// Create a separate auth client without interceptors to avoid circular dependencies
	authClient := v1connect.NewAuthServiceClient(httpClient, url)
	return func(ctx context.Context) (string, error) {
		req := connect.NewRequest(&v1pb.LoginRequest{
			Email:    email,
			Password: password,
		})

		resp, err := authClient.Login(ctx, req)
		if err != nil {
			return "", errors.Wrapf(err, "failed to login")
		}

		return resp.Msg.Token, nil
	}
}

func (c *Client) CheckRelease(ctx context.Context, r *v1pb.CheckReleaseRequest) (*v1pb.CheckReleaseResponse, error) {
	resp, err := c.releaseClient.CheckRelease(ctx, connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check release")
	}
	return resp.Msg, nil
}

func (c *Client) CreateRelease(ctx context.Context, project string, r *v1pb.Release) (*v1pb.Release, error) {
	req := connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent:  project,
		Release: r,
	})
	resp, err := c.releaseClient.CreateRelease(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create release")
	}
	return resp.Msg, nil
}

func (c *Client) GetPlan(ctx context.Context, planName string) (*v1pb.Plan, error) {
	resp, err := c.planClient.GetPlan(ctx,
		connect.NewRequest(&v1pb.GetPlanRequest{
			Name: planName,
		}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan")
	}
	return resp.Msg, nil
}

func (c *Client) CreatePlan(ctx context.Context, project string, r *v1pb.Plan) (*v1pb.Plan, error) {
	req := connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project,
		Plan:   r,
	})
	resp, err := c.planClient.CreatePlan(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create plan")
	}
	return resp.Msg, nil
}

func (c *Client) GetRollout(ctx context.Context, rolloutName string) (*v1pb.Rollout, error) {
	resp, err := c.rolloutClient.GetRollout(ctx,
		connect.NewRequest(&v1pb.GetRolloutRequest{
			Name: rolloutName,
		}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rollout")
	}
	return resp.Msg, nil
}

func (c *Client) CreateRollout(ctx context.Context, r *v1pb.CreateRolloutRequest) (*v1pb.Rollout, error) {
	resp, err := c.rolloutClient.CreateRollout(ctx, connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create rollout")
	}
	return resp.Msg, nil
}

func (c *Client) BatchRunTasks(ctx context.Context, r *v1pb.BatchRunTasksRequest) (*v1pb.BatchRunTasksResponse, error) {
	resp, err := c.rolloutClient.BatchRunTasks(ctx, connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to batch run tasks")
	}
	return resp.Msg, nil
}

func (c *Client) ListTaskRuns(ctx context.Context, r *v1pb.ListTaskRunsRequest) (*v1pb.ListTaskRunsResponse, error) {
	resp, err := c.rolloutClient.ListTaskRuns(ctx, connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list task runs")
	}
	return resp.Msg, nil
}

func (c *Client) ListAllTaskRuns(ctx context.Context, rolloutName string) (*v1pb.ListTaskRunsResponse, error) {
	request := &v1pb.ListTaskRunsRequest{
		Parent: rolloutName + "/stages/-/tasks/-",
	}
	return c.ListTaskRuns(ctx, request)
}

func (c *Client) BatchCancelTaskRuns(ctx context.Context, r *v1pb.BatchCancelTaskRunsRequest) (*v1pb.BatchCancelTaskRunsResponse, error) {
	resp, err := c.rolloutClient.BatchCancelTaskRuns(ctx, connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to batch cancel task runs")
	}
	return resp.Msg, nil
}

func (c *Client) GetActuatorInfo(ctx context.Context) (*v1pb.ActuatorInfo, error) {
	resp, err := c.actuatorClient.GetActuatorInfo(ctx,
		connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get actuator info")
	}
	return resp.Msg, nil
}

// Close cleans up resources used by the Client
func (c *Client) Close() error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	code := connect.CodeOf(err)
	switch code {
	case connect.CodeUnavailable,
		connect.CodeDeadlineExceeded,
		connect.CodeResourceExhausted:
		return true
	case connect.CodeUnknown:
		// Some network errors may be wrapped as Unknown
		msg := err.Error()
		return strings.Contains(msg, "connection refused") ||
			strings.Contains(msg, "connection reset") ||
			strings.Contains(msg, "timeout")
	default:
		return false
	}
}

// tokenManager handles thread-safe token storage
type tokenManager struct {
	mu    sync.RWMutex
	token string
}

func newTokenManager() *tokenManager {
	return &tokenManager{}
}

func (tm *tokenManager) setToken(token string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = token
}

func (tm *tokenManager) getToken() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.token
}

// unifiedInterceptor combines authentication and retry logic
type unifiedInterceptor struct {
	tokenMgr        *tokenManager
	maxAttempts     int
	initialInterval time.Duration
	maxInterval     time.Duration
	tokenRefresher  func(ctx context.Context) (string, error)
	refreshGroup    singleflight.Group
}

func newUnifiedInterceptor(config *RetryConfig, tokenRefresher func(ctx context.Context) (string, error)) *unifiedInterceptor {
	ui := &unifiedInterceptor{
		tokenMgr:       newTokenManager(),
		tokenRefresher: tokenRefresher,
	}

	if config != nil && config.MaxAttempts > 1 {
		ui.maxAttempts = config.MaxAttempts
		ui.initialInterval = config.InitialInterval
		ui.maxInterval = config.MaxInterval
	} else {
		// Default retry config for auth errors
		ui.maxAttempts = 2
		ui.initialInterval = 1 * time.Second
		ui.maxInterval = 1 * time.Second
	}

	return ui
}

func (ui *unifiedInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		var lastErr error
		interval := ui.initialInterval

		for attempt := 0; attempt < ui.maxAttempts; attempt++ {
			if attempt > 0 {
				// Check if context is cancelled before retrying
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}

				// Wait before retry
				timer := time.NewTimer(interval)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}

				// Exponential backoff
				interval *= 2
				if interval > ui.maxInterval {
					interval = ui.maxInterval
				}
			}

			// Add auth header
			if req.Spec().IsClient {
				if token := ui.tokenMgr.getToken(); token != "" {
					req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
				}
			}

			resp, err := next(ctx, req)
			if err == nil {
				return resp, nil
			}

			lastErr = err
			code := connect.CodeOf(err)

			// Handle authentication errors
			if code == connect.CodeUnauthenticated && ui.tokenRefresher != nil {
				refreshErr := ui.reauthenticateOnce(ctx)
				if refreshErr != nil {
					return nil, errors.Wrap(refreshErr, "failed to login")
				}
				// Don't count auth retry against attempt limit
				attempt--
				continue
			}

			// Check if error is retryable
			if !isRetryableError(err) {
				return nil, err
			}
		}

		return nil, lastErr
	}
}

func (ui *unifiedInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if token := ui.tokenMgr.getToken(); token != "" {
			conn.RequestHeader().Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
		return conn
	})
}

func (*unifiedInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, conn)
	})
}

func (ui *unifiedInterceptor) reauthenticateOnce(ctx context.Context) error {
	// Use singleflight to ensure only one refresh happens at a time
	// Multiple concurrent requests will wait for the same refresh operation
	_, err, _ := ui.refreshGroup.Do("refresh", func() (any, error) {
		return nil, ui.doReauthenticate(ctx)
	})
	return err
}

func (ui *unifiedInterceptor) doReauthenticate(ctx context.Context) error {
	if ui.tokenRefresher == nil {
		return errors.New("no token refresher available")
	}

	// Retry token refresh with exponential backoff for transient errors
	var lastErr error
	interval := ui.initialInterval
	maxAttempts := 3 // Fixed retry attempts for auth refresh

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Check context before retry
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Wait before retry
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			// Exponential backoff
			interval *= 2
			if interval > ui.maxInterval {
				interval = ui.maxInterval
			}
		}

		token, err := ui.tokenRefresher(ctx)
		if err == nil {
			// Update the token manager with the new token
			ui.tokenMgr.setToken(token)
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
	}

	return lastErr
}
