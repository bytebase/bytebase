package milvus

import (
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_MILVUS, parseStatements)
	base.RegisterSplitterFunc(storepb.Engine_MILVUS, SplitSQL)
	base.RegisterQueryValidator(storepb.Engine_MILVUS, validateQuery)
}

func parseStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	parsed := make([]base.ParsedStatement, 0, len(stmts))
	for _, stmt := range stmts {
		if stmt.Empty {
			parsed = append(parsed, base.ParsedStatement{Statement: stmt})
			continue
		}
		parsed = append(parsed, base.ParsedStatement{
			Statement: stmt,
			AST: &base.ANTLRAST{
				StartPosition: stmt.Start,
			},
		})
	}
	return parsed, nil
}

func validateQuery(statement string) (bool, bool, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return false, false, err
	}

	hasNonEmpty := false
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}
		hasNonEmpty = true
		lower := strings.ToLower(strings.TrimSpace(stmt.Text))
		switch {
		case strings.HasPrefix(lower, "select"),
			strings.HasPrefix(lower, "show"),
			strings.HasPrefix(lower, "desc "),
			strings.HasPrefix(lower, "describe "),
			strings.HasPrefix(lower, "with "),
			strings.HasPrefix(lower, "explain select"),
			strings.HasPrefix(lower, "explain with"):
			continue
		default:
			return false, false, nil
		}
	}

	if !hasNonEmpty {
		return false, false, errors.New("empty statement")
	}
	return true, true, nil
}
