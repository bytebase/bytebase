package tests

import (
	"bytes"
	"fmt"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
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

// syncSheet syncs sheets with project.
func (ctl *controller) syncSheet(projectID int) error {
	_, err := ctl.post(fmt.Sprintf("/project/%d/sync-sheet", projectID), nil)
	return err
}
