package v1

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func isReadOnlyStatementForAccessGrant(ctx context.Context, engine storepb.Engine, statement string) (bool, error) {
	readOnly, _, err := parserbase.ValidateSQLForEditor(engine, statement)
	if err != nil {
		return false, err
	}
	if !readOnly {
		return false, nil
	}

	if !shouldClassifyStatementByQuerySpan(engine) {
		return true, nil
	}
	spans, err := getQuerySpansForStatement(ctx, engine, statement)
	if err != nil {
		return false, err
	}
	if len(spans) == 0 {
		return false, nil
	}
	for _, span := range spans {
		if span == nil {
			return false, nil
		}
		switch span.Type {
		case parserbase.Select, parserbase.SelectInfoSchema, parserbase.Explain:
		case parserbase.QueryTypeUnknown, parserbase.DDL, parserbase.DML:
			return false, nil
		default:
			return false, nil
		}
	}
	return true, nil
}

func getQuerySpansForStatement(ctx context.Context, engine storepb.Engine, statement string) ([]*parserbase.QuerySpan, error) {
	statements, err := parserbase.SplitMultiSQL(engine, statement)
	if err != nil {
		statements = []parserbase.Statement{{Text: statement}}
	}
	return parserbase.GetQuerySpan(ctx, parserbase.GetQuerySpanContext{}, engine, statements, "", "", false)
}

func shouldClassifyStatementByQuerySpan(engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_MONGODB, storepb.Engine_ELASTICSEARCH:
		return true
	default:
		return false
	}
}
