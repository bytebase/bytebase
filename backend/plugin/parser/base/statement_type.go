package base

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// StatementTypeWithPosition pairs a classified statement type with its 1-based end line and SQL text.
type StatementTypeWithPosition struct {
	Type storepb.StatementType
	Line int
	Text string
}

// CollectUniqueStatementTypes returns the deduplicated set of statement types classified from asts.
func CollectUniqueStatementTypes(asts []AST, classify func(AST) (storepb.StatementType, error)) ([]storepb.StatementType, error) {
	sqlTypeSet := make(map[storepb.StatementType]bool, len(asts))
	for _, a := range asts {
		typ, err := classify(a)
		if err != nil {
			return nil, err
		}
		sqlTypeSet[typ] = true
	}
	var sqlTypes []storepb.StatementType
	for sqlType := range sqlTypeSet {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

// CollectStatementTypesWithPosition returns statement types paired with 1-based end lines and SQL text.
func CollectStatementTypesWithPosition(asts []AST, classify func(AST) ([]storepb.StatementType, string, bool, error)) ([]StatementTypeWithPosition, error) {
	if len(asts) == 0 {
		return []StatementTypeWithPosition{}, nil
	}
	var results []StatementTypeWithPosition
	for _, a := range asts {
		types, text, ok, err := classify(a)
		if err != nil {
			return nil, err
		}
		if !ok || len(types) == 0 {
			continue
		}
		endLine := 0
		if start := a.ASTStartPosition(); start != nil {
			endLine = int(start.Line)
		}
		if text != "" {
			endLine += strings.Count(text, "\n")
		}
		for _, typ := range types {
			results = append(results, StatementTypeWithPosition{
				Type: typ,
				Line: endLine,
				Text: text,
			})
		}
	}
	return results, nil
}

// ExtractTypesFromPositions projects the Type field out of StatementTypeWithPosition entries.
func ExtractTypesFromPositions(items []StatementTypeWithPosition) []storepb.StatementType {
	types := make([]storepb.StatementType, len(items))
	for i, item := range items {
		types[i] = item.Type
	}
	return types
}
