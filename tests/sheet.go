package tests

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

// createSheet creates a sheet.
func (ctl *controller) createSheet(sheetCreate api.SheetCreate) (*api.Sheet, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &sheetCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal sheetCreate")
	}

	body, err := ctl.post("/sheet", buf)
	if err != nil {
		return nil, err
	}

	sheet := new(api.Sheet)
	if err = jsonapi.UnmarshalPayload(body, sheet); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal sheet response")
	}
	return sheet, nil
}

// listMySheets lists caller's sheets.
func (ctl *controller) listMySheets() ([]*api.Sheet, error) {
	params := map[string]string{}
	body, err := ctl.get("/sheet/my", params)
	if err != nil {
		return nil, err
	}

	var sheets []*api.Sheet
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Sheet)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get sheet response")
	}
	for _, p := range ps {
		sheet, ok := p.(*api.Sheet)
		if !ok {
			return nil, errors.Errorf("fail to convert sheet")
		}
		sheets = append(sheets, sheet)
	}
	return sheets, nil
}

// syncSheet syncs sheets with project.
func (ctl *controller) syncSheet(projectID int) error {
	_, err := ctl.post(fmt.Sprintf("/project/%d/sync-sheet", projectID), nil)
	return err
}
