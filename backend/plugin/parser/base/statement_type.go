package base

import (
	"errors"
	"strings"

	omnianalysis "github.com/bytebase/omni/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// OmniStatementType maps an omni analysis.StatementType to storepb.StatementType.
// Unspecified or unrecognized values return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED.
func OmniStatementType(t omnianalysis.StatementType) storepb.StatementType {
	return omniStatementTypeMap[t]
}

// OmniStatementTypes converts a slice of omni statement types, omitting entries
// that map to storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED.
func OmniStatementTypes(types []omnianalysis.StatementType) []storepb.StatementType {
	var result []storepb.StatementType
	for _, t := range types {
		if value := OmniStatementType(t); value != storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED {
			result = append(result, value)
		}
	}
	return result
}

var omniStatementTypeMap = map[omnianalysis.StatementType]storepb.StatementType{
	omnianalysis.CreateDatabase:  storepb.StatementType_CREATE_DATABASE,
	omnianalysis.CreateTable:     storepb.StatementType_CREATE_TABLE,
	omnianalysis.CreateView:      storepb.StatementType_CREATE_VIEW,
	omnianalysis.CreateIndex:     storepb.StatementType_CREATE_INDEX,
	omnianalysis.CreateSequence:  storepb.StatementType_CREATE_SEQUENCE,
	omnianalysis.CreateSchema:    storepb.StatementType_CREATE_SCHEMA,
	omnianalysis.CreateFunction:  storepb.StatementType_CREATE_FUNCTION,
	omnianalysis.CreateTrigger:   storepb.StatementType_CREATE_TRIGGER,
	omnianalysis.CreateProcedure: storepb.StatementType_CREATE_PROCEDURE,
	omnianalysis.CreateEvent:     storepb.StatementType_CREATE_EVENT,
	omnianalysis.CreateExtension: storepb.StatementType_CREATE_EXTENSION,
	omnianalysis.CreateType:      storepb.StatementType_CREATE_TYPE,
	omnianalysis.DropDatabase:    storepb.StatementType_DROP_DATABASE,
	omnianalysis.DropTable:       storepb.StatementType_DROP_TABLE,
	omnianalysis.DropView:        storepb.StatementType_DROP_VIEW,
	omnianalysis.DropIndex:       storepb.StatementType_DROP_INDEX,
	omnianalysis.DropSequence:    storepb.StatementType_DROP_SEQUENCE,
	omnianalysis.DropSchema:      storepb.StatementType_DROP_SCHEMA,
	omnianalysis.DropFunction:    storepb.StatementType_DROP_FUNCTION,
	omnianalysis.DropTrigger:     storepb.StatementType_DROP_TRIGGER,
	omnianalysis.DropProcedure:   storepb.StatementType_DROP_PROCEDURE,
	omnianalysis.DropEvent:       storepb.StatementType_DROP_EVENT,
	omnianalysis.DropExtension:   storepb.StatementType_DROP_EXTENSION,
	omnianalysis.DropType:        storepb.StatementType_DROP_TYPE,
	omnianalysis.AlterDatabase:   storepb.StatementType_ALTER_DATABASE,
	omnianalysis.AlterTable:      storepb.StatementType_ALTER_TABLE,
	omnianalysis.AlterView:       storepb.StatementType_ALTER_VIEW,
	omnianalysis.AlterSequence:   storepb.StatementType_ALTER_SEQUENCE,
	omnianalysis.AlterEvent:      storepb.StatementType_ALTER_EVENT,
	omnianalysis.AlterType:       storepb.StatementType_ALTER_TYPE,
	omnianalysis.AlterIndex:      storepb.StatementType_ALTER_INDEX,
	omnianalysis.Truncate:        storepb.StatementType_TRUNCATE,
	omnianalysis.Rename:          storepb.StatementType_RENAME,
	omnianalysis.RenameIndex:     storepb.StatementType_RENAME_INDEX,
	omnianalysis.RenameSchema:    storepb.StatementType_RENAME_SCHEMA,
	omnianalysis.RenameSequence:  storepb.StatementType_RENAME_SEQUENCE,
	omnianalysis.Comment:         storepb.StatementType_COMMENT,
	omnianalysis.Insert:          storepb.StatementType_INSERT,
	omnianalysis.Update:          storepb.StatementType_UPDATE,
	omnianalysis.Delete:          storepb.StatementType_DELETE,
	omnianalysis.Merge:           storepb.StatementType_MERGE,
	omnianalysis.Execute:         storepb.StatementType_EXECUTE,
}

// StatementTypeWithPosition pairs a classified statement type with its 1-based end line and SQL text.
type StatementTypeWithPosition struct {
	Type storepb.StatementType
	Line int
	Text string
}

// collectUniqueStatementTypes returns the deduplicated set of statement types classified from asts.
func collectUniqueStatementTypes(asts []AST, classify func(AST) (storepb.StatementType, error)) ([]storepb.StatementType, error) {
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

// CollectOmniStatementTypes extracts each engine's omni AST node via getNode,
// returning typeErr on mismatch, and collects the unique mapped StatementTypes.
func CollectOmniStatementTypes[N any](
	asts []AST,
	getNode func(AST) (N, bool),
	typeErr string,
	classify func(N) omnianalysis.StatementType,
) ([]storepb.StatementType, error) {
	return collectUniqueStatementTypes(asts, func(a AST) (storepb.StatementType, error) {
		node, ok := getNode(a)
		if !ok {
			return 0, errors.New(typeErr)
		}
		return OmniStatementType(classify(node)), nil
	})
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
