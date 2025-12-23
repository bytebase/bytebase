package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type UserResult struct {
	Roles []string `json:"roles"`
}

// xpackInfoResponse represents the response from GET /_xpack API.
type xpackInfoResponse struct {
	Features struct {
		Security struct {
			Available bool `json:"available"`
			Enabled   bool `json:"enabled"`
		} `json:"security"`
	} `json:"features"`
}

// isSecurityEnabled checks if Elasticsearch security features are available and enabled
// by calling the /_xpack info API. Returns (enabled, error).
// If the /_xpack endpoint is not available (OSS build), returns (false, nil).
func (d *Driver) isSecurityEnabled(ctx context.Context) (bool, error) {
	if d.typedClient != nil {
		resp, err := esapi.XPackInfoRequest{}.Do(ctx, d.typedClient)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		// OSS builds don't have /_xpack endpoint
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			slog.Info("X-Pack not available (likely OSS build), security features disabled")
			return false, nil
		}

		if resp.IsError() {
			body, _ := io.ReadAll(resp.Body)
			return false, errors.Errorf("failed to get X-Pack info: %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, errors.Wrap(err, "failed to read X-Pack info response")
		}

		var info xpackInfoResponse
		if err := json.Unmarshal(body, &info); err != nil {
			return false, errors.Wrap(err, "failed to parse X-Pack info response")
		}

		return info.Features.Security.Available && info.Features.Security.Enabled, nil
	}

	// For basicAuthClient, call /_xpack directly
	resp, err := d.basicAuthClient.Do("GET", []byte("/_xpack"), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// OSS builds don't have /_xpack endpoint
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		slog.Info("X-Pack not available (likely OSS build), security features disabled")
		return false, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Wrap(err, "failed to read X-Pack info response")
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("failed to get X-Pack info: %d: %s", resp.StatusCode, string(body))
	}

	var info xpackInfoResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return false, errors.Wrap(err, "failed to parse X-Pack info response")
	}

	return info.Features.Security.Available && info.Features.Security.Enabled, nil
}

func (d *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
	// AWS IAM authentication doesn't use internal users - skip role fetching
	if d.config.DataSource.GetAuthenticationType() == storepb.DataSource_AWS_RDS_IAM {
		return nil, nil
	}

	var bytes []byte

	if d.isOpenSearch && d.opensearchClient != nil {
		// OpenSearch uses a different security plugin architecture.
		// Check if security plugin is available by calling the API and handling errors gracefully.
		resp, err := d.basicAuthClient.Do("GET", []byte("/_plugins/_security/api/internalusers"), nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		bytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read OpenSearch users response body")
		}

		// OpenSearch without security plugin returns 400 or 404
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			slog.Info("OpenSearch security plugin not available, skipping role discovery")
			return nil, nil
		}

		if resp.StatusCode != http.StatusOK {
			return nil, errors.Errorf("failed to get OpenSearch users: unexpected status code %d: %s", resp.StatusCode, string(bytes))
		}
	} else {
		// Elasticsearch: check X-Pack security availability first
		securityEnabled, err := d.isSecurityEnabled(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check X-Pack security status")
		}
		if !securityEnabled {
			slog.Info("Elasticsearch security is not enabled, skipping role discovery")
			return nil, nil
		}

		// Security is enabled, proceed to fetch users
		if d.typedClient != nil {
			resp, err := esapi.SecurityGetUserRequest{Pretty: true}.Do(ctx, d.typedClient)
			if err != nil {
				return nil, err
			}

			bytes, err = readBytesAndClose(resp)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get Elasticsearch users")
			}
		} else {
			resp, err := d.basicAuthClient.Do("GET", []byte("/_security/user"), nil)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			bytes, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read Elasticsearch users response body")
			}

			if resp.StatusCode != http.StatusOK {
				return nil, errors.Errorf("failed to get Elasticsearch users: unexpected status code %d: %s", resp.StatusCode, string(bytes))
			}
		}
	}

	var instanceRoles []*storepb.InstanceRole
	results := map[string]UserResult{}
	if err := json.Unmarshal(bytes, &results); err != nil {
		// Include response body to help debug parsing issues
		bodyPreview := string(bytes)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "..."
		}
		slog.Error("failed to parse users response", log.BBError(err), slog.String("response", bodyPreview))
		return instanceRoles, nil
	}

	for name, userResult := range results {
		privileges, err := d.getUserPrivileges(name)
		if err != nil {
			return nil, err
		}
		attribute := fmt.Sprintf("[%s]: %s", strings.Join(userResult.Roles, ", "), privileges)
		instanceRoles = append(instanceRoles, &storepb.InstanceRole{
			Name:      name,
			Attribute: &attribute,
		})
	}
	return instanceRoles, nil
}

func (d *Driver) getUserPrivileges(usrName string) (string, error) {
	if d.isOpenSearch && d.opensearchClient != nil {
		return "", nil
	} else if d.typedClient != nil {
		header := http.Header{}
		header.Add("Authorization", d.config.DataSource.GetAuthenticationPrivateKey())
		header.Add("es-security-runas-user", usrName)
		resp, err := esapi.SecurityGetUserPrivilegesRequest{Header: header}.Do(context.Background(), d.typedClient)
		if err != nil {
			return "", err
		}

		bytes, err := readBytesAndClose(resp)
		if err != nil {
			return "", errors.Wrap(err, "failed to get user privileges")
		}
		return string(bytes), nil
	}
	return "", nil
}
