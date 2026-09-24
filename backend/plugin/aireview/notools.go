package aireview

import (
	"context"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// NoTools is the Tools of a review that reads nothing beyond the prompt: the
// model judges from the target facts and the statements alone.
type NoTools struct{}

// Definitions implements Tools.
func (NoTools) Definitions() []*v1pb.AIChatToolDefinition {
	return nil
}

// Call implements Tools. The loop never reaches it, since it offers no tool.
func (NoTools) Call(_ context.Context, name string, _ string) (string, error) {
	return "", errors.Errorf("tool %q does not exist: this review has no tools", name)
}
