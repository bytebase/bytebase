package elasticsearch

import (
	"context"
	"io"
)

// Dump() is not applicable to Elasticsearch.
func (*Driver) Dump(_ context.Context, _ io.Writer) error {
	return nil
}
