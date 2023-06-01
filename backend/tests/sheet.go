package tests

import (
	"fmt"
)

// syncSheet syncs sheets with project.
func (ctl *controller) syncSheet(projectID int) error {
	_, err := ctl.post(fmt.Sprintf("/project/%d/sync-sheet", projectID), nil)
	return err
}
