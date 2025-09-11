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

func (d *Driver) getInstanceRoles() ([]*storepb.InstanceRole, error) {
	var bytes []byte

	if d.isOpenSearch && d.opensearchClient != nil {
		resp, err := d.basicAuthClient.Do("GET", []byte("/_plugins/_security/api/internalusers"), nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		bytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read OpenSearch users response body")
		}

		// Check HTTP status code
		if resp.StatusCode != http.StatusOK {
			// Include response body for debugging
			return nil, errors.Errorf("failed to get OpenSearch users: unexpected status code %d: %s", resp.StatusCode, string(bytes))
		}
	} else if d.typedClient != nil {
		resp, err := esapi.SecurityGetUserRequest{Pretty: true}.Do(context.Background(), d.typedClient)
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

		// Check HTTP status code
		if resp.StatusCode != http.StatusOK {
			// Include response body for debugging
			return nil, errors.Errorf("failed to get Elasticsearch users: unexpected status code %d: %s", resp.StatusCode, string(bytes))
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
