package segment

import (
	"context"

	"github.com/bytebase/bytebase/store"
)

// Reporter is the API message for segment reporter
type Reporter interface {
	Report(ctx context.Context, store *store.Store, segment *segment) error
}
