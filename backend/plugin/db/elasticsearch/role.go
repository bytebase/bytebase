package elasticsearch

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func (d *Driver) ListRole(ctx context.Context) ([]*db.DatabaseRoleMessage, error) {
	resp, err := esapi.SecurityGetRoleRequest{}.Do(ctx, d.typedClient)
	if err != nil {
		return nil, err
	}

	bytes, err := readBytesAndClose(resp)
	if err != nil {
		return nil, err
	}

	dbRoleMessages := []*db.DatabaseRoleMessage{}

	roles := map[string]any{}
	if err := json.Unmarshal(bytes, &roles); err != nil {
		return nil, err
	}

	for roleName, roleInfo := range roles {
		roleInfoStr, ok := roleInfo.(string)
		if !ok {
			return nil, errors.New("type assertion fails: roleInfo")
		}
		dbRoleMessages = append(dbRoleMessages, &db.DatabaseRoleMessage{
			Name:      roleName,
			Attribute: &roleInfoStr,
		})
	}

	return dbRoleMessages, nil
}

func (d *Driver) FindRole(ctx context.Context, name string) (*db.DatabaseRoleMessage, error) {
	resp, err := esapi.SecurityGetRoleRequest{Name: []string{name}}.Do(ctx, d.typedClient)
	if err != nil {
		return nil, err
	}

	bytes, err := readBytesAndClose(resp)
	if err != nil {
		return nil, err
	}

	dbRoleMessage := &db.DatabaseRoleMessage{}

	roles := map[string]any{}
	if err := json.Unmarshal(bytes, &roles); err != nil {
		return nil, err
	}

	for roleName, roleInfo := range roles {
		roleInfoStr, ok := roleInfo.(string)
		if !ok {
			return nil, errors.New("type assertion fails: roleInfo")
		}
		dbRoleMessage.Name = roleName
		dbRoleMessage.Attribute = &roleInfoStr
	}

	return dbRoleMessage, nil
}

func (d *Driver) CreateRole(ctx context.Context, msg *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	resp, err := esapi.SecurityPutRoleRequest{Name: msg.Name}.Do(ctx, d.typedClient)
	if err != nil || resp.StatusCode != 200 {
		return nil, err
	}
	return nil, nil
}

func (d *Driver) DeleteRole(ctx context.Context, name string) error {
	resp, err := esapi.SecurityDeleteRoleRequest{Name: name}.Do(ctx, d.typedClient)
	if err != nil || resp.StatusCode != 200 {
		return err
	}
	return nil
}

func (d *Driver) UpdateRole(ctx context.Context, _ string, msg *db.DatabaseRoleUpsertMessage) (*db.DatabaseRoleMessage, error) {
	dbRoleMsg, err := d.CreateRole(ctx, msg)
	if err != nil {
		return nil, err
	}
	return dbRoleMsg, nil
}

type UserResult struct {
	Roles []string `json:"roles"`
}

func (d *Driver) getUsersWithRoles() (map[string]UserResult, error) {
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

	return results, nil
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
