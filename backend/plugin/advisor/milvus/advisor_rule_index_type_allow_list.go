package milvus

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	createIndexRE = regexp.MustCompile(`(?is)^\s*create\s+index\s+on\s+([A-Za-z0-9_]+)\s+field\s+([A-Za-z0-9_]+)(?:\s+with\s+(\{.*\}))?\s*;?\s*$`)

	binaryMetricTypes = []string{"HAMMING", "JACCARD", "TANIMOTO", "SUBSTRUCTURE", "SUPERSTRUCTURE"}
	floatMetricTypes  = []string{"L2", "IP", "COSINE"}
)

func init() {
	advisor.Register(storepb.Engine_MILVUS, storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST, &IndexTypeAllowListAdvisor{})
}

// IndexTypeAllowListAdvisor checks index type allowlist and basic metric compatibility for Milvus create index statements.
type IndexTypeAllowListAdvisor struct{}

func (*IndexTypeAllowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	allowList := normalizeStringSet(checkCtx.Rule.GetStringArrayPayload().GetList())
	var adviceList []*storepb.Advice

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.Empty {
			continue
		}
		matches := createIndexRE.FindStringSubmatch(stmt.Text)
		if len(matches) == 0 {
			continue
		}
		indexType, metricType := extractIndexAndMetricType(matches[3])
		if indexType == "" {
			// No explicit index type in the statement payload.
			continue
		}

		if len(allowList) > 0 && !allowList[indexType] {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.IndexTypeNotAllowed.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("Milvus index type %q is not in the allowlist", indexType),
				StartPosition: stmt.Start,
			})
			continue
		}

		if metricType == "" {
			continue
		}
		if !isMetricCompatible(indexType, metricType) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          advisorcode.IndexTypeNotAllowed.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("Milvus index type %q is incompatible with metric type %q", indexType, metricType),
				StartPosition: stmt.Start,
			})
		}
	}
	return adviceList, nil
}

func extractIndexAndMetricType(rawJSON string) (string, string) {
	rawJSON = strings.TrimSpace(rawJSON)
	if rawJSON == "" {
		return "", ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return "", ""
	}

	indexType := strings.ToUpper(strings.TrimSpace(toString(payload["indexType"])))
	metricType := strings.ToUpper(strings.TrimSpace(toString(payload["metricType"])))

	if params, ok := payload["params"].(map[string]any); ok && metricType == "" {
		metricType = strings.ToUpper(strings.TrimSpace(toString(params["metricType"])))
	}
	if indexParams, ok := payload["indexParams"]; ok {
		extractedIndex, extractedMetric := extractFromIndexParams(indexParams)
		if indexType == "" {
			indexType = extractedIndex
		}
		if metricType == "" {
			metricType = extractedMetric
		}
	}
	return indexType, metricType
}

func extractFromIndexParams(v any) (string, string) {
	switch value := v.(type) {
	case map[string]any:
		return strings.ToUpper(strings.TrimSpace(toString(value["indexType"]))), strings.ToUpper(strings.TrimSpace(toString(value["metricType"])))
	case []any:
		for _, item := range value {
			if m, ok := item.(map[string]any); ok {
				indexType, metricType := extractFromIndexParams(m)
				if indexType != "" || metricType != "" {
					return indexType, metricType
				}
			}
		}
	}
	return "", ""
}

func isMetricCompatible(indexType string, metricType string) bool {
	isBinaryIndex := strings.HasPrefix(indexType, "BIN_")
	if isBinaryIndex {
		return slices.Contains(binaryMetricTypes, metricType)
	}
	if slices.Contains(binaryMetricTypes, metricType) {
		return false
	}
	// Default float-vector index family compatibility.
	return slices.Contains(floatMetricTypes, metricType)
}

func normalizeStringSet(input []string) map[string]bool {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]bool, len(input))
	for _, item := range input {
		value := strings.ToUpper(strings.TrimSpace(item))
		if value != "" {
			result[value] = true
		}
	}
	return result
}

func toString(v any) string {
	value := strings.TrimSpace(fmt.Sprintf("%v", v))
	if value == "<nil>" {
		return ""
	}
	return value
}

func compactStatement(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}
