package command

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

//nolint:forbidigo
var protojsonUnmarshaler = protojson.UnmarshalOptions{DiscardUnknown: true}

// Client is the API message for Bytebase API Client.
type Client struct {
	client *http.Client

	url   string
	token string
}

// NewClient returns the new Bytebase API client.
func NewClient(url, serviceAccount, serviceAccountSecret string) (*Client, error) {
	c := Client{
		client: &http.Client{Timeout: 120 * time.Second},
		url:    url,
	}

	if err := c.login(serviceAccount, serviceAccountSecret); err != nil {
		return nil, errors.Wrapf(err, "failed to login")
	}

	return &c, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}

func (c *Client) login(email, password string) error {
	r := &v1pb.LoginRequest{
		Email:    email,
		Password: password,
	}
	rb, err := protojson.Marshal(r)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal")
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/auth/login", c.url), bytes.NewReader(rb))
	if err != nil {
		return errors.Wrapf(err, "failed to create request")
	}

	body, err := c.doRequest(req)
	if err != nil {
		return errors.Wrapf(err, "failed to login")
	}

	resp := &v1pb.LoginResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return errors.Wrapf(err, "failed to unmarshal")
	}
	c.token = resp.Token

	return nil
}

func (c *Client) checkRelease(project string, r *v1pb.CheckReleaseRequest) (*v1pb.CheckReleaseResponse, error) {
	rb, err := protojson.Marshal(r)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/%s/releases:check", c.url, project), bytes.NewReader(rb))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check release")
	}

	resp := &v1pb.CheckReleaseResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) createRelease(project string, r *v1pb.Release) (*v1pb.Release, error) {
	rb, err := protojson.Marshal(r)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/%s/releases", c.url, project), bytes.NewReader(rb))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create release")
	}

	resp := &v1pb.Release{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) getPlan(planName string) (*v1pb.Plan, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/%s", c.url, planName), nil)
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan")
	}
	resp := &v1pb.Plan{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) createPlan(project string, r *v1pb.Plan) (*v1pb.Plan, error) {
	rb, err := protojson.Marshal(r)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/%s/plans", c.url, project), bytes.NewReader(rb))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create plan")
	}

	resp := &v1pb.Plan{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) runPlanChecks(r *v1pb.RunPlanChecksRequest) (*v1pb.RunPlanChecksResponse, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/%s:runPlanChecks", c.url, r.Name), nil)
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run plan checks")
	}
	resp := &v1pb.RunPlanChecksResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) listPlanCheckRuns(r *v1pb.ListPlanCheckRunsRequest) (*v1pb.ListPlanCheckRunsResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/%s/planCheckRuns", c.url, r.Parent), nil)
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list plan check runs")
	}
	resp := &v1pb.ListPlanCheckRunsResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
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
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/%s", c.url, rolloutName), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get rollout")
	}

	resp := &v1pb.Rollout{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) createRollout(r *v1pb.CreateRolloutRequest) (*v1pb.Rollout, error) {
	rb, err := protojson.Marshal(r.Rollout)
	if err != nil {
		return nil, err
	}
	a := fmt.Sprintf("%s/v1/%s/rollouts", c.url, r.Parent)
	query := url.Values{}
	if r.ValidateOnly {
		query.Set("validateOnly", "true")
	}
	if r.Target != nil {
		query.Set("target", *r.Target)
	}
	if len(query) > 0 {
		a += "?" + query.Encode()
	}

	req, err := http.NewRequest("POST", a, bytes.NewReader(rb))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create rollout")
	}

	resp := &v1pb.Rollout{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) batchRunTasks(r *v1pb.BatchRunTasksRequest) (*v1pb.BatchRunTasksResponse, error) {
	rb, err := protojson.Marshal(r)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/%s/tasks:batchRun", c.url, r.Parent), bytes.NewReader(rb))
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to batch run tasks")
	}
	resp := &v1pb.BatchRunTasksResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) listTaskRuns(r *v1pb.ListTaskRunsRequest) (*v1pb.ListTaskRunsResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/%s/taskRuns", c.url, r.Parent), nil)
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list task runs")
	}
	resp := &v1pb.ListTaskRunsResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
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
	rb, err := protojson.Marshal(r)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/%s/taskRuns:batchCancel", c.url, r.Parent), bytes.NewReader(rb))
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to batch cancel task runs")
	}
	resp := &v1pb.BatchCancelTaskRunsResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) getActuatorInfo() (*v1pb.ActuatorInfo, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/actuator/info", c.url), nil)
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get actuator info")
	}
	resp := &v1pb.ActuatorInfo{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
