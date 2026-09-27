package reviewrun

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
)

// AIReviewExecutor is the AI review: a model judges the change against the
// natural-language AI review policy.
//
// Not implemented yet: the evaluation lands with the AI review design. A
// claimed AI run must still reach a terminal status, so it fails honestly
// instead of reporting a vacuous DONE, which would carry the "every unit was
// evaluated" meaning the gate later relies on.
type AIReviewExecutor struct{}

// NewAIReviewExecutor creates the AI review executor.
func NewAIReviewExecutor() *AIReviewExecutor {
	return &AIReviewExecutor{}
}

// RunOnce implements Executor.
func (*AIReviewExecutor) RunOnce(_ context.Context, _ string, _ int64) ([]*store.IssueCommentMessage, error) {
	return nil, errors.New("AI review is not implemented yet")
}
