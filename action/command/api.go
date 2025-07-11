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

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// authInterceptor implements connect.Interceptor to add authentication headers
type authInterceptor struct {
	mu    sync.RWMutex
	token string
}

func (a *authInterceptor) setToken(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = token
}

func (a *authInterceptor) getToken() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.token
}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Spec().IsClient {
			token := a.getToken()
			if token != "" {
				req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
			}
		}
		return next(ctx, req)
	})
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		token := a.getToken()
		if token != "" {
			conn.RequestHeader().Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
		return conn
	})
}

func (*authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, conn)
	})
}

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

	// Authentication
	token       string
	interceptor *authInterceptor

	// Connect RPC service clients
	authClient     v1connect.AuthServiceClient
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
	// Apply defaults for zero values
	if opts.PageSize == 0 {
		opts.PageSize = 100
	}
	if opts.PageSize > 1000 {
		opts.PageSize = 1000 // Server-side limit
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

	authInterceptor := &authInterceptor{}

	// Build interceptor chain
	interceptorList := []connect.Interceptor{authInterceptor}

	// Add retry interceptor if configured
	if opts.RetryConfig != nil && opts.RetryConfig.MaxAttempts > 1 {
		retryInterceptor := &retryInterceptor{
			maxAttempts:     opts.RetryConfig.MaxAttempts,
			initialInterval: opts.RetryConfig.InitialInterval,
			maxInterval:     opts.RetryConfig.MaxInterval,
		}
		interceptorList = append(interceptorList, retryInterceptor)
	}

	interceptors := connect.WithInterceptors(interceptorList...)

	c := Client{
		httpClient:     httpClient,
		url:            url,
		interceptor:    authInterceptor,
		options:        opts,
		authClient:     v1connect.NewAuthServiceClient(httpClient, url, interceptors),
		releaseClient:  v1connect.NewReleaseServiceClient(httpClient, url, interceptors),
		planClient:     v1connect.NewPlanServiceClient(httpClient, url, interceptors),
		rolloutClient:  v1connect.NewRolloutServiceClient(httpClient, url, interceptors),
		actuatorClient: v1connect.NewActuatorServiceClient(httpClient, url, interceptors),
	}

	if err := c.login(context.Background(), serviceAccount, serviceAccountSecret); err != nil {
		return nil, errors.Wrapf(err, "failed to login")
	}

	return &c, nil
}

func (c *Client) login(ctx context.Context, email, password string) error {
	req := connect.NewRequest(&v1pb.LoginRequest{
		Email:    email,
		Password: password,
	})

	resp, err := c.authClient.Login(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "failed to login")
	}

	c.token = resp.Msg.Token
	c.interceptor.setToken(resp.Msg.Token)

	return nil
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

func (c *Client) RunPlanChecks(ctx context.Context, r *v1pb.RunPlanChecksRequest) (*v1pb.RunPlanChecksResponse, error) {
	resp, err := c.planClient.RunPlanChecks(ctx, connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run plan checks")
	}
	return resp.Msg, nil
}

func (c *Client) ListPlanCheckRuns(ctx context.Context, r *v1pb.ListPlanCheckRunsRequest) (*v1pb.ListPlanCheckRunsResponse, error) {
	resp, err := c.planClient.ListPlanCheckRuns(ctx, connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list plan check runs")
	}
	return resp.Msg, nil
}

func (c *Client) ListAllPlanCheckRuns(ctx context.Context, planName string) (*v1pb.ListPlanCheckRunsResponse, error) {
	resp := &v1pb.ListPlanCheckRunsResponse{}
	request := &v1pb.ListPlanCheckRunsRequest{
		Parent:    planName,
		PageSize:  c.options.PageSize,
		PageToken: "",
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		listResp, err := c.ListPlanCheckRuns(ctx, request)
		if err != nil {
			return nil, err
		}
		resp.PlanCheckRuns = append(resp.PlanCheckRuns, listResp.PlanCheckRuns...)
		if listResp.NextPageToken == "" {
			break
		}
		request.PageToken = listResp.NextPageToken
	}
	return resp, nil
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
	resp := &v1pb.ListTaskRunsResponse{}
	request := &v1pb.ListTaskRunsRequest{
		Parent:    rolloutName + "/stages/-/tasks/-",
		PageSize:  c.options.PageSize,
		PageToken: "",
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		listResp, err := c.ListTaskRuns(ctx, request)
		if err != nil {
			return nil, err
		}
		resp.TaskRuns = append(resp.TaskRuns, listResp.TaskRuns...)
		if listResp.NextPageToken == "" {
			break
		}
		request.PageToken = listResp.NextPageToken
	}
	return resp, nil
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

// hasErrorMessage checks if the error message contains specific text
func hasErrorMessage(err error, msg string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), msg)
}

// retryInterceptor implements retry logic for transient errors
type retryInterceptor struct {
	maxAttempts     int
	initialInterval time.Duration
	maxInterval     time.Duration
}

func (r *retryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		var lastErr error
		interval := r.initialInterval

		for attempt := 0; attempt < r.maxAttempts; attempt++ {
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
				if interval > r.maxInterval {
					interval = r.maxInterval
				}
			}

			resp, err := next(ctx, req)
			if err == nil {
				return resp, nil
			}

			lastErr = err

			// Check if error is retryable
			if !isRetryableError(err) {
				return nil, err
			}
		}

		return nil, lastErr
	}
}

func (*retryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	// No retry for streaming operations
	return next
}

func (*retryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
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
