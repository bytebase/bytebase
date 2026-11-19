package aireview

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// CheckSetting reports why a project gets no AI SQL review, or nil when it
// does. The review has a switch of its own because it sends the statements of
// a change to the provider and can send the schema of its targets, which
// turning AI on for the SQL editor does not agree to. projectEnabled is the
// project's own switch, which replaces the workspace's; nil follows the
// workspace. AI itself is on or off for the whole workspace either way.
func CheckSetting(setting *storepb.AISetting, projectEnabled *bool) error {
	if !setting.GetEnabled() {
		return errors.New("AI is not enabled in the workspace setting")
	}
	if projectEnabled != nil {
		if !*projectEnabled {
			return errors.New("AI SQL review is turned off for the project")
		}
		return nil
	}
	if !setting.GetSqlReviewEnabled() {
		return errors.New("AI SQL review is not enabled in the workspace AI setting")
	}
	return nil
}
