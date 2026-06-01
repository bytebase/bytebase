package plsql

import (
	"context"
	"fmt"
	"strings"

	oracleast "github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_ORACLE, GenerateRestoreSQL)
}

const (
	maxCommentLength = 1000
)

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractSQL(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	node, err := findFirstOracleDML(originalSQL)
	if err != nil {
		return "", err
	}
	if node == nil {
		return "", errors.New("no DML statement found in extracted SQL")
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}
	return doGenerate(ctx, rCtx, sqlForComment, node, backupItem)
}

func findFirstOracleDML(statement string) (oracleast.StmtNode, error) {
	node, err := findFirstOracleDMLOnce(statement)
	if err == nil {
		return node, nil
	}
	if !strings.Contains(statement, "\n") {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	lines := strings.Split(statement, "\n")
	for start := range lines {
		var candidate strings.Builder
		for end := start; end < len(lines); end++ {
			line := strings.TrimSpace(lines[end])
			if line == "" && candidate.Len() == 0 {
				continue
			}
			if candidate.Len() > 0 {
				candidate.WriteByte('\n')
			}
			candidate.WriteString(lines[end])
			node, candidateErr := findFirstOracleDMLOnce(candidate.String())
			if candidateErr != nil {
				continue
			}
			if node != nil {
				return node, nil
			}
		}
	}
	return nil, errors.Wrap(err, "failed to parse statement")
}

func findFirstOracleDMLOnce(statement string) (oracleast.StmtNode, error) {
	list, err := ParsePLSQLOmni(statement)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, nil
	}
	for _, item := range list.Items {
		raw, ok := item.(*oracleast.RawStmt)
		if !ok || raw.Stmt == nil {
			continue
		}
		switch raw.Stmt.(type) {
		case *oracleast.UpdateStmt, *oracleast.DeleteStmt:
			return raw.Stmt, nil
		default:
		}
	}
	return nil, nil
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, node oracleast.StmtNode, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}
	_, targetDatabase, err := common.GetInstanceDatabaseID(backupItem.TargetTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get target database ID for %s", backupItem.TargetTable.Database)
	}

	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Errorf("GetDatabaseMetadataFunc is required")
	}

	_, metadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}

	schemaMetadata := metadata.GetSchemaMetadata("")
	if schemaMetadata == nil {
		return "", errors.Errorf("no schema metadata for %s", sourceDatabase)
	}

	tableMetadata := schemaMetadata.GetTable(backupItem.SourceTable.Table)
	if tableMetadata == nil {
		return "", errors.Errorf("no table metadata for %s.%s", sourceDatabase, backupItem.SourceTable.Table)
	}

	g := &generator{
		ctx:              ctx,
		rCtx:             rCtx,
		backupDatabase:   targetDatabase,
		backupTable:      backupItem.TargetTable.Table,
		originalDatabase: sourceDatabase,
		originalTable:    backupItem.SourceTable.Table,
		pk:               tableMetadata.GetPrimaryKey(),
		table:            tableMetadata,
		isFirst:          true,
	}
	if err := g.generate(node); err != nil {
		return "", err
	}
	return fmt.Sprintf("/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, g.result), nil
}

type generator struct {
	backupDatabase   string
	backupTable      string
	originalDatabase string
	originalTable    string
	pk               *model.IndexMetadata
	table            *model.TableMetadata

	isFirst bool
	ctx     context.Context
	rCtx    base.RestoreContext
	result  string
}

func (g *generator) generate(node oracleast.StmtNode) error {
	if !g.isFirst {
		return nil
	}
	g.isFirst = false

	switch n := node.(type) {
	case *oracleast.DeleteStmt:
		g.generateDelete()
	case *oracleast.UpdateStmt:
		return g.generateUpdate(n)
	default:
		return errors.Errorf("unexpected statement type: %T", node)
	}
	return nil
}

func (g *generator) generateDelete() {
	columnList := quoteOracleColumns(g.restorableColumns())
	g.result = fmt.Sprintf(`INSERT INTO "%s"."%s" (%s) SELECT %s FROM "%s"."%s";`, g.originalDatabase, g.originalTable, columnList, columnList, g.backupDatabase, g.backupTable)
}

func (g *generator) restorableColumns() []string {
	var columns []string
	for _, column := range g.table.GetProto().GetColumns() {
		if column.GetGeneration() != nil {
			continue
		}
		columns = append(columns, column.Name)
	}
	return columns
}

func quoteOracleColumns(columns []string) string {
	var quotedColumns []string
	for _, column := range columns {
		quotedColumns = append(quotedColumns, fmt.Sprintf(`"%s"`, column))
	}
	return strings.Join(quotedColumns, ", ")
}

func disjoint(a []string, b map[string]bool) bool {
	for _, item := range a {
		if _, ok := b[item]; ok {
			return false
		}
	}
	return true
}

func (g *generator) findDisjointUniqueKey(columns []string) ([]string, error) {
	columnMap := make(map[string]bool)
	for _, column := range columns {
		columnMap[column] = true
	}
	if g.pk != nil {
		if disjoint(g.pk.GetProto().Expressions, columnMap) {
			return g.pk.GetProto().Expressions, nil
		}
	}
	for _, index := range g.table.GetProto().Indexes {
		if index.Primary {
			continue
		}
		if !index.Unique {
			continue
		}
		if strings.Contains(index.Type, "FUNCTION-BASED") {
			continue
		}
		if disjoint(index.Expressions, columnMap) {
			return index.Expressions, nil
		}
	}
	return nil, errors.Errorf("no disjoint unique key found for %s.%s", g.originalDatabase, g.originalTable)
}

func (g *generator) generateUpdate(stmt *oracleast.UpdateStmt) error {
	updateColumns := extractOmniUpdateColumns(stmt)
	uk, err := g.findDisjointUniqueKey(updateColumns)
	if err != nil {
		return err
	}

	var buf strings.Builder
	if _, err := fmt.Fprintf(&buf, "MERGE INTO \"%s\".\"%s\" t\nUSING \"%s\".\"%s\" b\n  ON(", g.originalDatabase, g.originalTable, g.backupDatabase, g.backupTable); err != nil {
		return err
	}
	for i, column := range uk {
		if i > 0 {
			if _, err := fmt.Fprint(&buf, " AND"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(&buf, " t.\"%s\" = b.\"%s\"", column, column); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(&buf, ")\nWHEN MATCHED THEN\n  UPDATE SET"); err != nil {
		return err
	}
	for i, field := range updateColumns {
		if i > 0 {
			if _, err := fmt.Fprint(&buf, ","); err != nil {
				return err
			}
		}
		// The field is written by user and no need to escape.
		if _, err := fmt.Fprintf(&buf, " t.\"%s\" = b.\"%s\"", field, field); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(&buf, "\nWHEN NOT MATCHED THEN\n INSERT ("); err != nil {
		return err
	}
	for i, column := range g.restorableColumns() {
		if i > 0 {
			if _, err := fmt.Fprint(&buf, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(&buf, "\"%s\"", column); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(&buf, ") VALUES ("); err != nil {
		return err
	}
	for i, column := range g.restorableColumns() {
		if i > 0 {
			if _, err := fmt.Fprint(&buf, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(&buf, "b.\"%s\"", column); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(&buf, ");"); err != nil {
		return err
	}
	g.result = buf.String()
	return nil
}

func extractOmniUpdateColumns(stmt *oracleast.UpdateStmt) []string {
	var result []string
	if stmt == nil || stmt.SetClauses == nil {
		return result
	}
	for _, item := range stmt.SetClauses.Items {
		clause, ok := item.(*oracleast.SetClause)
		if !ok {
			continue
		}
		if clause.Column != nil {
			result = append(result, clause.Column.Column)
		}
		for _, column := range omniColumnRefList(clause.Columns) {
			result = append(result, column.Column)
		}
	}
	return result
}

func omniColumnRefList(list *oracleast.List) []*oracleast.ColumnRef {
	if list == nil {
		return nil
	}
	result := make([]*oracleast.ColumnRef, 0, len(list.Items))
	for _, item := range list.Items {
		if column, ok := item.(*oracleast.ColumnRef); ok {
			result = append(result, column)
		}
	}
	return result
}

func extractSQL(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.New("backup item is nil")
	}

	list, err := SplitSQL(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to split SQL")
	}

	start := 0
	end := len(list) - 1
	for i, item := range list {
		if equalOrLess(item.Start, backupItem.StartPosition) {
			start = i
		}
	}

	for i := len(list) - 1; i >= 0; i-- {
		if equalOrGreater(list[i].Start, backupItem.EndPosition) {
			end = i
		}
	}

	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}

	var result []string
	for i := start; i <= end; i++ {
		containsSourceTable := false
		tables, err := prepareTransformation(sourceDatabase, list[i].Text)
		if err != nil {
			if err == errNoBackupableDML {
				continue
			}
			return "", errors.Wrap(err, "failed to prepare transformation")
		}
		for _, table := range tables {
			if table.table.Schema == sourceDatabase && table.table.Table == backupItem.SourceTable.Table {
				containsSourceTable = true
				break
			}
		}
		if containsSourceTable {
			result = append(result, normalizeExtractedRestoreSQL(list[i].Text))
		}
	}
	return strings.Join(result, "\n"), nil
}

func normalizeExtractedRestoreSQL(statement string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(statement), ";"))
}

func equalOrLess(a, b *storepb.Position) bool {
	if a.Line < b.Line {
		return true
	}
	if a.Line == b.Line && a.Column <= b.Column {
		return true
	}
	return false
}

func equalOrGreater(a, b *storepb.Position) bool {
	if a.Line > b.Line {
		return true
	}
	if a.Line == b.Line && a.Column >= b.Column {
		return true
	}
	return false
}
