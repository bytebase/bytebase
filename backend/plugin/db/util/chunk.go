package util

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ChunkedSQLScript splits a SQL script into chunks.
func ChunkedSQLScript(slice []base.SingleSQL, n int) ([][]base.SingleSQL, error) {
	if n <= 0 {
		return nil, errors.Errorf("invalid number of chunks: %d", n)
	}

	length := len(slice)
	chunkSize := length / n
	remainder := length % n

	chunks := make([][]base.SingleSQL, n)
	start := 0

	for i := 0; i < n; i++ {
		end := start + chunkSize
		if i < remainder {
			end++
		}

		if start >= end {
			// Empty chunk.
			// We can stop here because the remaining chunks will also be empty.
			break
		}
		chunks[i] = slice[start:end]
		start = end
	}

	return chunks, nil
}

// ConcatChunk is the optimization in the case that we have a 100MB text.
func ConcatChunk(chunk []base.SingleSQL) (string, error) {
	if len(chunk) == 1 {
		return chunk[0].Text, nil
	}
	var chunkBuf strings.Builder
	for _, sql := range chunk {
		if _, err := chunkBuf.WriteString(sql.Text); err != nil {
			return "", errors.Wrapf(err, "failed to write chunk buffer")
		}
	}
	return chunkBuf.String(), nil
}
