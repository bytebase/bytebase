package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v7/esapi"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// getInstanceRoles fetches users/roles for display purposes only.
// Fails open on any error since most deployments don't use internal users.
func (d *Driver) getInstanceRoles(ctx context.Context) ([]*storepb.InstanceRole, error) {
	var resp *http.Response
	var err error

	if d.isOpenSearch && d.opensearchClient != nil {
		resp, err = d.basicAuthClient.Do("GET", []byte("/_plugins/_security/api/internalusers"), nil)
	} else if d.typedClient != nil {
		var esResp *esapi.Response
		esResp, err = esapi.SecurityGetUserRequest{}.Do(ctx, d.typedClient)
		if esResp != nil {
			resp = &http.Response{StatusCode: esResp.StatusCode, Body: esResp.Body}
		}
	} else if d.basicAuthClient != nil {
		resp, err = d.basicAuthClient.Do("GET", []byte("/_security/user"), nil)
	}

	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}

	results := map[string]struct {
		Roles []string `json:"roles"`
	}{}
	if err := json.Unmarshal(bytes, &results); err != nil {
		return nil, nil
	}

	var roles []*storepb.InstanceRole
	for name, user := range results {
		attr := fmt.Sprintf("[%s]", strings.Join(user.Roles, ", "))
		roles = append(roles, &storepb.InstanceRole{Name: name, Attribute: &attr})
	}
	return roles, nil
}
