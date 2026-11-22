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
	return base.ConvertOffsetBackupStatements(statement, statements, func(s analysis.BackupStatement) (string, string, string, string, int, int) {
		return s.Statement, s.SourceSchema, s.SourceTableName, s.TargetTableName, s.Start, s.End
	}, true), nil
}
