package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

//nolint:forbidigo
var protojsonUnmarshaler = protojson.UnmarshalOptions{DiscardUnknown: true}

// client is the API message for Bytebase API client.
type client struct {
	client *http.Client

	url   string
	token string
}

// NewClient returns the new Bytebase API client.
func NewClient(url, email, password string) (*client, error) {
	c := client{
		client: &http.Client{Timeout: 10 * time.Second},
		url:    url,
	}

	if err := c.login(email, password); err != nil {
		return nil, err
	}

	return &c, nil
}
func (c *client) doRequest(req *http.Request) ([]byte, error) {
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

func (c *client) login(email, password string) error {
	r := &v1pb.LoginRequest{
		Email:    email,
		Password: password,
	}
	rb, err := protojson.Marshal(r)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/auth/login", c.url), strings.NewReader(string(rb)))
	if err != nil {
		return err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return errors.Wrapf(err, "failed to login")
	}

	resp := v1pb.LoginResponse{}
	if err := protojsonUnmarshaler.Unmarshal(body, &resp); err != nil {
		return err
	}
	c.token = resp.Token

	return nil
}
