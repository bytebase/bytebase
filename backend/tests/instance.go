package tests

import (
	"fmt"
	"reflect"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func (ctl *controller) getInstanceMigrationHistory(instanceID int, find db.MigrationHistoryFind) ([]*api.MigrationHistory, error) {
	params := make(map[string]string)
	if find.Database != nil {
		params["database"] = *find.Database
	}
	if find.Version != nil {
		params["version"] = *find.Version
	}
	if find.Limit != nil {
		params["limit"] = fmt.Sprintf("%d", *find.Limit)
	}
	body, err := ctl.get(fmt.Sprintf("/instance/%d/migration/history", instanceID), params)
	if err != nil {
		return nil, err
	}

	var histories []*api.MigrationHistory
	hs, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.MigrationHistory)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get migration history response")
	}
	for _, h := range hs {
		history, ok := h.(*api.MigrationHistory)
		if !ok {
			return nil, errors.Errorf("fail to convert migration history")
		}
		histories = append(histories, history)
	}
	return histories, nil
}

func (ctl *controller) getInstanceSDLMigrationHistory(instanceID int, historyID string) (*api.MigrationHistory, error) {
	params := make(map[string]string)
	params["sdl"] = "true"
	body, err := ctl.get(fmt.Sprintf("/instance/%d/migration/history/%s", instanceID, historyID), params)
	if err != nil {
		return nil, err
	}
	result := new(api.MigrationHistory)
	if err := jsonapi.UnmarshalPayload(body, result); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get migration history response")
	}
	return result, nil
}
