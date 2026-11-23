package pg

import (
	"context"
	"strings"

	"github.com/bytebase/omni/pg/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_POSTGRES, GenerateRestoreSQL)
}

// ExtractPrependStatement returns the first SET role or SET search_path
// statement in statement, trimmed and terminated with a semicolon, or "" if
// none is present.
func ExtractPrependStatement(statement string) (string, error) {
	stmts, err := ParsePg(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}
	prepend := analysis.PrependStatements(stmts)
	if prepend == "" {
		return "", nil
	}
	return prepend + ";", nil
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	stmts, err := base.ParseForRestore(statement, backupItem, ParsePg)
	if err != nil {
		return "", err
	}

	posMapper := base.NewByteOffsetPositionMapper(statement)
	dmls := base.FilterDMLsInRange(analysis.BackedUpDMLs(stmts), backupItem, func(dml analysis.BackedUpDML) (*storepb.Position, *storepb.Position) {
		startPos := posMapper.Position(dml.Start)
		startPos.Column-- // 0-based to match backup convention
		return startPos, posMapper.Position(dml.End)
	})

	var sqls []string
	for _, dml := range dmls {
		// Statement text without trailing semicolon/whitespace.
		sqls = append(sqls, strings.TrimRight(strings.TrimSpace(dml.Text), ";"))
	}
	originalSQL := strings.Join(sqls, ";\n")
	if len(originalSQL) == 0 {
		return "", errors.New("no original SQL")
	}
	if len(dmls) == 0 {
		return "", base.StatementNotFoundError(backupItem)
	}

	sourceDatabase, err := base.ParseBackupSourceDatabase(backupItem)
	if err != nil {
		return "", err
	}
	tableMeta, err := base.ResolveRestoreTableMetadata(ctx, rCtx, sourceDatabase, backupItem.SourceTable, "public", func(db string) error {
		return errors.Errorf("database metadata not found for %s", db)
	})
	if err != nil {
		return "", err
	}

	return analysis.RestoreSQL(dmls[0].Node, analysis.RestoreOptions{
		OriginalSchema:    tableMeta.Schema,
		OriginalTable:     backupItem.SourceTable.Table,
		BackupSchema:      backupItem.TargetTable.Schema,
		BackupTable:       backupItem.TargetTable.Table,
		Table:             tableMeta.Table,
		PrimaryKey:        tableMeta.PrimaryKey,
		OriginalSQL:       base.FormatRestoreComment(originalSQL),
		PrependStatements: analysis.PrependStatements(stmts),
	})
}
