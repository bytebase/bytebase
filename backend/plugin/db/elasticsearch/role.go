package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/esapi"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type UserResult struct {
	Roles []string `json:"roles"`
}

func (d *Driver) getInstanceRoles() ([]*storepb.InstanceRole, error) {
	resp, err := esapi.SecurityGetUserRequest{Pretty: true}.Do(context.Background(), d.typedClient)
	if err != nil {
		return nil, err
	}

	bytes, err := readBytesAndClose(resp)
	if err != nil {
		return nil, err
	}

	results := map[string]UserResult{}
	if err := json.Unmarshal(bytes, &results); err != nil {
		return nil, err
	}

	var instanceRoles []*storepb.InstanceRole
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
	header := http.Header{}
	header.Add("Authorization", d.config.AuthenticationPrivateKey)
	header.Add("es-security-runas-user", usrName)
	resp, err := esapi.SecurityGetUserPrivilegesRequest{Header: header}.Do(context.Background(), d.typedClient)
	if err != nil {
		return "", err
	}

	bytes, err := readBytesAndClose(resp)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
