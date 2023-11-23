package util

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ChunkedSQLScript splits a SQL script into chunks.
func ChunkedSQLScript(script []base.SingleSQL, maxChunksCount int) ([][]base.SingleSQL, error) {
	var result [][]base.SingleSQL

	if maxChunksCount <= 0 {
		return nil, errors.New("maxChunksCount must be greater than 0")
	}

	if len(script) == 0 {
		return result, nil
	}

	roundDown := len(script) / maxChunksCount
	rest := len(script) % maxChunksCount

	// We have len(script) sqls, and we want to split it into no more than maxChunksCount chunks.
	// len(script) = roundDown * maxChunksCount + rest
	//             = rest * (roundDown + 1)  + (maxChunksCount - rest) * roundDown
	// So the first $rest chunks will have $(roundDown + 1) sqls, and the rest $(maxChunksCount) chunks will have $roundDown sqls.

	for i, sql := range script {
		if i < rest*(roundDown+1) {
			if i%(roundDown+1) == 0 {
				result = append(result, []base.SingleSQL{})
			}
			result[len(result)-1] = append(result[len(result)-1], sql)
			continue
		}

		if (i-rest*(roundDown+1))%roundDown == 0 {
			result = append(result, []base.SingleSQL{})
		}
		result[len(result)-1] = append(result[len(result)-1], sql)
	}

	return result, nil
}
