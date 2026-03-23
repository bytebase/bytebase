package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	maxCommentLength = 1000
)

func init() {
	base.RegisterGenerateRestoreSQL(storepb.Engine_POSTGRES, GenerateRestoreSQL)
}

func GenerateRestoreSQL(ctx context.Context, rCtx base.RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	originalSQL, err := extractStatement(statement, backupItem)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}

	if len(originalSQL) == 0 {
		return "", errors.Errorf("no original SQL")
	}

	// Find the matching DML statement node at the backup position.
	matchingNode, err := findStatementAtPosition(statement, backupItem)
	if err != nil {
		return "", err
	}
	if matchingNode == nil {
		return "", errors.Errorf("could not find statement at position (line %d:%d - %d:%d)",
			backupItem.StartPosition.Line, backupItem.StartPosition.Column,
			backupItem.EndPosition.Line, backupItem.EndPosition.Column)
	}

	sqlForComment, truncated := common.TruncateString(originalSQL, maxCommentLength)
	if truncated {
		sqlForComment += "..."
	}

	prependStatements, err := getPrependStatements(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to get prepend statements")
	}

	return doGenerate(ctx, rCtx, sqlForComment, matchingNode, backupItem, prependStatements)
}

func findStatementAtPosition(statement string, backupItem *storepb.PriorBackupDetail_Item) (ast.Node, error) {
	stmts, err := ParsePg(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		switch stmt.AST.(type) {
		case *ast.UpdateStmt, *ast.DeleteStmt:
			nodeLoc := ast.NodeLoc(stmt.AST)
			startPos := ByteOffsetToRunePosition(statement, nodeLoc.Start)
			startPos.Column-- // 0-based to match backup convention
			endPos := ByteOffsetToRunePosition(statement, nodeLoc.End)
			if inRange(startPos, endPos, backupItem.StartPosition, backupItem.EndPosition) {
				return stmt.AST, nil
			}
		default:
		}
	}
	return nil, nil
}

func doGenerate(ctx context.Context, rCtx base.RestoreContext, sqlForComment string, node ast.Node, backupItem *storepb.PriorBackupDetail_Item, prependStatements string) (string, error) {
	_, sourceDatabase, err := common.GetInstanceDatabaseID(backupItem.SourceTable.Database)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get source database ID for %s", backupItem.SourceTable.Database)
	}

	if rCtx.GetDatabaseMetadataFunc == nil {
		return "", errors.Errorf("GetDatabaseMetadataFunc is required")
	}

	_, metadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}

	if metadata == nil {
		return "", errors.Errorf("database metadata not found for %s", sourceDatabase)
	}

	schema := backupItem.SourceTable.Schema
	if schema == "" {
		schema = "public"
	}
	schemaMetadata := metadata.GetSchemaMetadata(schema)
	if schemaMetadata == nil {
		return "", errors.Errorf("schema metadata not found for %s", schema)
	}

	tableMetadata := schemaMetadata.GetTable(backupItem.SourceTable.Table)
	if tableMetadata == nil {
		return "", errors.Errorf("table metadata not found for %s.%s", schema, backupItem.SourceTable.Table)
	}

	backupSchema := backupItem.TargetTable.Schema
	backupTable := backupItem.TargetTable.Table
	originalSchema := schema
	originalTable := backupItem.SourceTable.Table

	var result string
	switch n := node.(type) {
	case *ast.DeleteStmt:
		result = fmt.Sprintf(`INSERT INTO "%s"."%s" SELECT * FROM "%s"."%s";`, originalSchema, originalTable, backupSchema, backupTable)
	case *ast.UpdateStmt:
		fields := extractSetFieldNames(n)
		uk, err := findDisjointUniqueKey(tableMetadata, fields)
		if err != nil {
			return "", err
		}

		var buf strings.Builder
		if _, err := fmt.Fprintf(&buf, `INSERT INTO "%s"."%s" SELECT * FROM "%s"."%s" ON CONFLICT ON CONSTRAINT "%s" DO UPDATE SET `, originalSchema, originalTable, backupSchema, backupTable, uk); err != nil {
			return "", errors.Wrapf(err, "failed to generate update statement")
		}
		for i, field := range fields {
			if i > 0 {
				if _, err := fmt.Fprint(&buf, ", "); err != nil {
					return "", errors.Wrapf(err, "failed to generate update statement")
				}
			}
			if _, err := fmt.Fprintf(&buf, `"%s" = EXCLUDED."%s"`, field, field); err != nil {
				return "", errors.Wrapf(err, "failed to generate update statement")
			}
		}
		if _, err := fmt.Fprint(&buf, `;`); err != nil {
			return "", errors.Wrapf(err, "failed to generate update statement")
		}
		result = buf.String()
	default:
		return "", errors.Errorf("unexpected statement type: %T", node)
	}

	if len(prependStatements) > 0 {
		// Ensure prependStatements ends with semicolon
		if !strings.HasSuffix(prependStatements, ";") {
			prependStatements += ";"
		}
		return fmt.Sprintf("%s\n/*\nOriginal SQL:\n%s\n*/\n%s", prependStatements, sqlForComment, result), nil
	}
	return fmt.Sprintf("/*\nOriginal SQL:\n%s\n*/\n%s", sqlForComment, result), nil
}

// extractSetFieldNames extracts the target column names from an UPDATE SET clause.
func extractSetFieldNames(stmt *ast.UpdateStmt) []string {
	if stmt.TargetList == nil {
		return nil
	}
	var fields []string
	for _, item := range stmt.TargetList.Items {
		rt, ok := item.(*ast.ResTarget)
		if !ok || rt.Name == "" {
			continue
		}
		// Take the last name component (the actual column name).
		// For "SET test.c1 = 1", Name="test", Indirection=["c1"] → want "c1".
		// For "SET c1 = 1", Name="c1", Indirection=nil → want "c1".
		// For "SET schema.col[1] = val", Indirection=["col", integer] → want "col".
		if rt.Indirection != nil && rt.Indirection.Len() > 0 {
			found := false
			for j := rt.Indirection.Len() - 1; j >= 0; j-- {
				if s, ok := rt.Indirection.Items[j].(*ast.String); ok {
					fields = append(fields, s.Str)
					found = true
					break
				}
			}
			if found {
				continue
			}
		}
		fields = append(fields, rt.Name)
	}
	return fields
}

func disjoint(a []string, b map[string]bool) bool {
	for _, item := range a {
		if _, ok := b[item]; ok {
			return false
		}
	}
	return true
}

func findDisjointUniqueKey(table *model.TableMetadata, fields []string) (string, error) {
	columnMap := make(map[string]bool)
	for _, field := range fields {
		columnMap[field] = true
	}
	pk := table.GetPrimaryKey()
	if pk != nil {
		if disjoint(pk.GetProto().Expressions, columnMap) {
			return pk.GetProto().Name, nil
		}
	}
	for _, index := range table.GetProto().Indexes {
		if index.Primary {
			continue
		}
		if !index.Unique {
			continue
		}
		if disjoint(index.Expressions, columnMap) {
			return index.Name, nil
		}
	}

	return "", errors.Errorf("no disjoint unique key found for table")
}

func extractStatement(statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	if backupItem == nil {
		return "", errors.Errorf("backup item is nil")
	}

	stmts, err := ParsePg(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse statement")
	}

	var sqls []string
	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		switch stmt.AST.(type) {
		case *ast.UpdateStmt, *ast.DeleteStmt:
			nodeLoc := ast.NodeLoc(stmt.AST)
			startPos := ByteOffsetToRunePosition(statement, nodeLoc.Start)
			startPos.Column-- // 0-based
			endPos := ByteOffsetToRunePosition(statement, nodeLoc.End)
			if inRange(startPos, endPos, backupItem.StartPosition, backupItem.EndPosition) {
				// Extract statement text without trailing semicolon/whitespace.
				text := strings.TrimRight(strings.TrimSpace(stmt.Text), ";")
				sqls = append(sqls, text)
			}
		default:
		}
	}

	return strings.Join(sqls, ";\n"), nil
}

func inRange(start, end, targetStart, targetEnd *storepb.Position) bool {
	if start.Line < targetStart.Line || (start.Line == targetStart.Line && start.Column < targetStart.Column) {
		return false
	}
	if end.Line > targetEnd.Line || (end.Line == targetEnd.Line && end.Column > targetEnd.Column) {
		return false
	}
	return true
}

func getPrependStatements(statement string) (string, error) {
	stmts, err := ParsePg(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	for _, stmt := range stmts {
		if stmt.Empty() {
			continue
		}
		vs, ok := stmt.AST.(*ast.VariableSetStmt)
		if !ok {
			continue
		}
		name := strings.ToLower(vs.Name)
		if name == "role" || name == "search_path" {
			text := strings.TrimRight(strings.TrimSpace(stmt.Text), ";")
			return text, nil
		}
	}

	return "", nil
}
