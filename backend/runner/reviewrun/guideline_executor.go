package reviewrun

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
)

// GuidelineExecutor is the natural-language guideline reviewer, performed by
// AI.
//
// Not implemented yet: the guideline evaluation lands with the AI review
// design. A claimed GUIDELINE run must still reach a terminal status, so it
// fails honestly instead of reporting a vacuous DONE — DONE would carry the
// "every unit was evaluated" meaning the gate later relies on.
type GuidelineExecutor struct{}

// NewGuidelineExecutor creates the guideline review executor.
func NewGuidelineExecutor() *GuidelineExecutor {
	return &GuidelineExecutor{}
}

// RunOnce implements Executor.
func (*GuidelineExecutor) RunOnce(_ context.Context, _ string, _ int64) ([]*store.IssueCommentMessage, error) {
	return nil, errors.New("guideline review is not implemented yet")
}
