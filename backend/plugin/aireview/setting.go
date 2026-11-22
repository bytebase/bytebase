package aireview

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// CheckSetting reports why the workspace AI setting does not allow an AI SQL
// review, or nil when it does. The review has a switch of its own because it
// sends the statements of a change to the provider and can send the schema of
// its targets, which turning AI on for the SQL editor does not agree to.
func CheckSetting(setting *storepb.AISetting) error {
	if !setting.GetEnabled() {
		return errors.New("AI is not enabled in the workspace setting")
	}
	if !setting.GetSqlReviewEnabled() {
		return errors.New("AI SQL review is not enabled in the workspace AI setting")
	}
	return nil
}
