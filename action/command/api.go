package command

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// authInterceptor implements connect.Interceptor to add authentication headers
type authInterceptor struct {
	token string
}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Spec().IsClient && a.token != "" {
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
		}
		return next(ctx, req)
	})
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if a.token != "" {
			conn.RequestHeader().Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
		}
		return conn
	})
}

func (*authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, conn)
	})
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
}

// NewClient returns the new Bytebase API client.
func NewClient(url, serviceAccount, serviceAccountSecret string) (*Client, error) {
	httpClient := &http.Client{Timeout: 120 * time.Second}
	authInterceptor := &authInterceptor{}
	interceptors := connect.WithInterceptors(authInterceptor)

	c := Client{
		httpClient:     httpClient,
		url:            url,
		interceptor:    authInterceptor,
		authClient:     v1connect.NewAuthServiceClient(httpClient, url, interceptors),
		releaseClient:  v1connect.NewReleaseServiceClient(httpClient, url, interceptors),
		planClient:     v1connect.NewPlanServiceClient(httpClient, url, interceptors),
		rolloutClient:  v1connect.NewRolloutServiceClient(httpClient, url, interceptors),
		actuatorClient: v1connect.NewActuatorServiceClient(httpClient, url, interceptors),
	}

	if err := c.login(serviceAccount, serviceAccountSecret); err != nil {
		return nil, errors.Wrapf(err, "failed to login")
	}

	return &c, nil
}

func (c *Client) login(email, password string) error {
	req := connect.NewRequest(&v1pb.LoginRequest{
		Email:    email,
		Password: password,
	})

	resp, err := c.authClient.Login(context.Background(), req)
	if err != nil {
		return errors.Wrapf(err, "failed to login")
	}

	c.token = resp.Msg.Token
	c.interceptor.token = resp.Msg.Token

	return nil
}

func (c *Client) checkRelease(_ string, r *v1pb.CheckReleaseRequest) (*v1pb.CheckReleaseResponse, error) {
	// Note: project is already included in r.Parent
	resp, err := c.releaseClient.CheckRelease(context.Background(), connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check release")
	}
	return resp.Msg, nil
}

func (c *Client) createRelease(project string, r *v1pb.Release) (*v1pb.Release, error) {
	req := connect.NewRequest(&v1pb.CreateReleaseRequest{
		Parent:  project,
		Release: r,
	})
	resp, err := c.releaseClient.CreateRelease(context.Background(), req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create release")
	}
	return resp.Msg, nil
}

func (c *Client) getPlan(planName string) (*v1pb.Plan, error) {
	resp, err := c.planClient.GetPlan(context.Background(),
		connect.NewRequest(&v1pb.GetPlanRequest{
			Name: planName,
		}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan")
	}
	return resp.Msg, nil
}

func (c *Client) createPlan(project string, r *v1pb.Plan) (*v1pb.Plan, error) {
	req := connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project,
		Plan:   r,
	})
	resp, err := c.planClient.CreatePlan(context.Background(), req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create plan")
	}
	return resp.Msg, nil
}

func (c *Client) runPlanChecks(r *v1pb.RunPlanChecksRequest) (*v1pb.RunPlanChecksResponse, error) {
	resp, err := c.planClient.RunPlanChecks(context.Background(), connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run plan checks")
	}
	return resp.Msg, nil
}

func (c *Client) listPlanCheckRuns(r *v1pb.ListPlanCheckRunsRequest) (*v1pb.ListPlanCheckRunsResponse, error) {
	resp, err := c.planClient.ListPlanCheckRuns(context.Background(), connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list plan check runs")
	}
	return resp.Msg, nil
}

func (c *Client) listAllPlanCheckRuns(planName string) (*v1pb.ListPlanCheckRunsResponse, error) {
	resp := &v1pb.ListPlanCheckRunsResponse{}
	request := &v1pb.ListPlanCheckRunsRequest{
		Parent:    planName,
		PageSize:  1000,
		PageToken: "",
	}
	for {
		listResp, err := c.listPlanCheckRuns(request)
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

func (c *Client) getRollout(rolloutName string) (*v1pb.Rollout, error) {
	resp, err := c.rolloutClient.GetRollout(context.Background(),
		connect.NewRequest(&v1pb.GetRolloutRequest{
			Name: rolloutName,
		}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rollout")
	}
	return resp.Msg, nil
}

func (c *Client) createRollout(r *v1pb.CreateRolloutRequest) (*v1pb.Rollout, error) {
	resp, err := c.rolloutClient.CreateRollout(context.Background(), connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create rollout")
	}
	return resp.Msg, nil
}

func (c *Client) batchRunTasks(r *v1pb.BatchRunTasksRequest) (*v1pb.BatchRunTasksResponse, error) {
	resp, err := c.rolloutClient.BatchRunTasks(context.Background(), connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to batch run tasks")
	}
	return resp.Msg, nil
}

func (c *Client) listTaskRuns(r *v1pb.ListTaskRunsRequest) (*v1pb.ListTaskRunsResponse, error) {
	resp, err := c.rolloutClient.ListTaskRuns(context.Background(), connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list task runs")
	}
	return resp.Msg, nil
}

func (c *Client) listAllTaskRuns(rolloutName string) (*v1pb.ListTaskRunsResponse, error) {
	resp := &v1pb.ListTaskRunsResponse{}
	request := &v1pb.ListTaskRunsRequest{
		Parent:    rolloutName + "/stages/-/tasks/-",
		PageSize:  1000,
		PageToken: "",
	}
	for {
		listResp, err := c.listTaskRuns(request)
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

func (c *Client) batchCancelTaskRuns(r *v1pb.BatchCancelTaskRunsRequest) (*v1pb.BatchCancelTaskRunsResponse, error) {
	resp, err := c.rolloutClient.BatchCancelTaskRuns(context.Background(), connect.NewRequest(r))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to batch cancel task runs")
	}
	return resp.Msg, nil
}

func (c *Client) getActuatorInfo() (*v1pb.ActuatorInfo, error) {
	resp, err := c.actuatorClient.GetActuatorInfo(context.Background(),
		connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get actuator info")
	}
	return resp.Msg, nil
}
