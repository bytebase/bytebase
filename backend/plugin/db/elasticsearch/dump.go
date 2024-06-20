package elasticsearch

import (
	"context"
	"io"
)

// Dump() is not applicable to Elasticsearch.
func (*Driver) Dump(_ context.Context, _ io.Writer) (string, error) {
	return "", nil
}
