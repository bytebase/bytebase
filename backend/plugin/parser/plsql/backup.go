package plsql

import (
	"context"

	"github.com/bytebase/omni/oracle/analysis"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterTransformDMLToSelect(store.Engine_ORACLE, TransformDMLToSelect)
}

// TransformDMLToSelect transforms DML statement to SELECT statement.
// For Oracle, we only consider the managed on schema mode.
func TransformDMLToSelect(_ context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	opts := analysis.BackupOptions{
		SourceDatabase: sourceDatabase,
		TargetDatabase: targetDatabase,
		TablePrefix:    tablePrefix,
	}
	if version, ok := tCtx.Version.(*Version); ok {
		opts.Version = version
	}
	statements, err := analysis.TransformDMLToSelect(statement, opts)
	if err != nil {
		return nil, err
	}
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	var result []base.BackupStatement
	for _, s := range statements {
		result = append(result, base.BackupStatement{
			Statement:       s.Statement,
			SourceSchema:    s.SourceSchema,
			SourceTableName: s.SourceTableName,
			TargetTableName: s.TargetTableName,
			StartPosition:   zeroBasedColumnPosition(positionMapper.Position(s.Start)),
			EndPosition:     zeroBasedColumnPosition(positionMapper.Position(s.End)),
		})
	}
	return result, nil
}

func zeroBasedColumnPosition(pos *store.Position) *store.Position {
	if pos == nil {
		return nil
	}
	column := pos.Column
	if column > 0 {
		column--
	}
	return &store.Position{
		Line:   pos.Line,
		Column: column,
	}
}
