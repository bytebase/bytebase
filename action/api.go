package main

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
		client: &http.Client{Timeout: 10 * time.Second},
		url:    url,
	}

	if err := c.login(serviceAccount, serviceAccountSecret); err != nil {
		return nil, err
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
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/auth/login", c.url), bytes.NewReader(rb))
	if err != nil {
		return err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return errors.Wrapf(err, "failed to login")
	}

	resp := &v1pb.LoginResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, resp); err != nil {
		return err
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

func (c *Client) previewPlan(project string, r *v1pb.PreviewPlanRequest) (*v1pb.PreviewPlanResponse, error) {
	rb, err := protojson.Marshal(r)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/%s:previewPlan", c.url, project), bytes.NewReader(rb))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to preview plan")
	}

	resp := &v1pb.PreviewPlanResponse{}
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

func (c *Client) getRollout(project, rolloutID string) (*v1pb.Rollout, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/%s/rollouts/%s", c.url, project, rolloutID), nil)
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

func (c *Client) createRollout(project string, r *v1pb.CreateRolloutRequest) (*v1pb.Rollout, error) {
	rb, err := protojson.Marshal(r.Rollout)
	if err != nil {
		return nil, err
	}
	a := fmt.Sprintf("%s/v1/%s/rollouts", c.url, project)
	if r.Target != nil {
		query := url.Values{}
		query.Set("target", *r.Target)
		a = a + "?" + query.Encode()
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
