package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	maxTableNameLength = 63
)

type StatementType int

const (
	StatementTypeUnknown StatementType = iota
	StatementTypeUpdate
	StatementTypeInsert
	StatementTypeDelete
)

type TableReference struct {
	Database      string
	Schema        string
	Table         string
	Alias         string
	StatementType StatementType
}

func (t *TableReference) String() string {
	if t.Database != "" {
		return fmt.Sprintf(`"%s"."%s"."%s"`, t.Database, t.Schema, t.Table)
	}

	if t.Schema != "" {
		return fmt.Sprintf(`"%s"."%s"`, t.Schema, t.Table)
	}

	return fmt.Sprintf(`"%s"`, t.Table)
}

type statementInfo struct {
	offset    int
	statement string
	node      ast.Node
	table     *TableReference
	startPos  *storepb.Position
	endPos    *storepb.Position
	fullSQL   string
}

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_POSTGRES, TransformDMLToSelect)
}

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, _ string, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(ctx, tCtx, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to prepare transformation")
	}

	return generateSQL(statementInfoList, targetSchema, tablePrefix)
}

func generateSQL(statementInfoList []statementInfo, targetSchema string, tablePrefix string) ([]base.BackupStatement, error) {
	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s", item.table.Schema, item.table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements on the same table.
	for key, list := range groupByTable {
		statementType := StatementTypeUnknown
		for _, item := range list {
			if statementType == StatementTypeUnknown {
				statementType = item.table.StatementType
			}
			if statementType != item.table.StatementType {
				return nil, errors.Errorf("The statement type is not the same for all statements on the same table %q", key)
			}
		}
	}

	var result []base.BackupStatement
	for key, list := range groupByTable {
		backupStatement, err := generateSQLForTable(list, targetSchema, tablePrefix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate SQL for table %q", key)
		}
		result = append(result, *backupStatement)
	}

	slices.SortFunc(result, func(a, b base.BackupStatement) int {
		if a.StartPosition.Line != b.StartPosition.Line {
			if a.StartPosition.Line < b.StartPosition.Line {
				return -1
			}
			return 1
		}
		if a.StartPosition.Column != b.StartPosition.Column {
			if a.StartPosition.Column < b.StartPosition.Column {
				return -1
			}
			return 1
		}
		if a.SourceTableName < b.SourceTableName {
			return -1
		}
		if a.SourceTableName > b.SourceTableName {
			return 1
		}
		return 0
	})

	return result, nil
}

func generateSQLForTable(statementInfoList []statementInfo, targetSchema string, tablePrefix string) (*base.BackupStatement, error) {
	table := statementInfoList[0].table

	targetTable := fmt.Sprintf("%s_%s_%s", tablePrefix, table.Table, table.Schema)
	targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
	var buf strings.Builder
	if _, err := fmt.Fprintf(&buf, `CREATE TABLE "%s"."%s" AS`+"\n", targetSchema, targetTable); err != nil {
		return nil, errors.Wrap(err, "failed to write to buffer")
	}

	// In PostgreSQL, WITH clause must appear once before the first SELECT.
	cteClause := extractCTE(statementInfoList[0].node, statementInfoList[0].fullSQL)
	if cteClause != "" {
		if _, err := fmt.Fprintf(&buf, "%s\n", cteClause); err != nil {
			return nil, errors.Wrap(err, "failed to write to buffer")
		}
	}

	for i, item := range statementInfoList {
		if i != 0 {
			if _, err := buf.WriteString("\n  UNION\n"); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		}
		if table.Alias != "" {
			if _, err := fmt.Fprintf(&buf, `  SELECT "%s".* `, table.Alias); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		} else {
			if _, err := fmt.Fprintf(&buf, `  SELECT %s.* `, table.String()); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		}

		if err := writeSuffixSelectClause(&buf, item.node, item.fullSQL); err != nil {
			return nil, errors.Wrap(err, "failed to write string with new line")
		}
	}

	if _, err := buf.WriteString(";"); err != nil {
		return nil, errors.Wrap(err, "failed to write to buffer")
	}

	return &base.BackupStatement{
		Statement:       buf.String(),
		SourceSchema:    table.Schema,
		SourceTableName: table.Table,
		TargetTableName: targetTable,
		StartPosition:   statementInfoList[0].startPos,
		EndPosition:     statementInfoList[len(statementInfoList)-1].endPos,
	}, nil
}

func extractCTE(node ast.Node, fullSQL string) string {
	var wc *ast.WithClause
	switch n := node.(type) {
	case *ast.UpdateStmt:
		wc = n.WithClause
	case *ast.DeleteStmt:
		wc = n.WithClause
	default:
	}
	if wc == nil {
		return ""
	}
	return extractNodeText(wc.Loc, fullSQL)
}

func writeSuffixSelectClause(buf *strings.Builder, node ast.Node, fullSQL string) error {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		return writeUpdateSuffix(buf, n, fullSQL)
	case *ast.DeleteStmt:
		return writeDeleteSuffix(buf, n, fullSQL)
	}
	return nil
}

func writeUpdateSuffix(buf *strings.Builder, n *ast.UpdateStmt, sql string) error {
	if n.Relation == nil {
		return nil
	}

	relText := extractNodeText(n.Relation.Loc, sql)
	if _, err := fmt.Fprintf(buf, "FROM %s", relText); err != nil {
		return errors.Wrap(err, "failed to write to buffer")
	}

	if n.FromClause != nil && n.FromClause.Len() > 0 {
		fromText := extractNodeText(ast.ListSpan(n.FromClause), sql)
		if fromText != "" {
			if _, err := fmt.Fprintf(buf, ", %s", fromText); err != nil {
				return errors.Wrap(err, "failed to write to buffer")
			}
		}
	}

	if n.WhereClause != nil {
		whereText := extractNodeText(ast.NodeLoc(n.WhereClause), sql)
		if whereText != "" {
			if _, err := fmt.Fprintf(buf, " WHERE %s", whereText); err != nil {
				return errors.Wrap(err, "failed to write to buffer")
			}
		}
	}
	return nil
}

func writeDeleteSuffix(buf *strings.Builder, n *ast.DeleteStmt, sql string) error {
	if n.Relation == nil {
		return nil
	}

	relText := extractNodeText(n.Relation.Loc, sql)
	if _, err := fmt.Fprintf(buf, "FROM %s", relText); err != nil {
		return errors.Wrap(err, "failed to write to buffer")
	}

	if n.UsingClause != nil && n.UsingClause.Len() > 0 {
		usingText := extractNodeText(ast.ListSpan(n.UsingClause), sql)
		if usingText != "" {
			if _, err := fmt.Fprintf(buf, ", %s", usingText); err != nil {
				return errors.Wrap(err, "failed to write to buffer")
			}
		}
	}

	if n.WhereClause != nil {
		whereText := extractNodeText(ast.NodeLoc(n.WhereClause), sql)
		if whereText != "" {
			if _, err := fmt.Fprintf(buf, " WHERE %s", whereText); err != nil {
				return errors.Wrap(err, "failed to write to buffer")
			}
		}
	}
	return nil
}

// extractNodeText extracts trimmed text from SQL using a Loc range.
func extractNodeText(loc ast.Loc, sql string) string {
	if loc.Start < 0 || loc.End < 0 || loc.Start >= loc.End || loc.End > len(sql) {
		return ""
	}
	return strings.TrimSpace(sql[loc.Start:loc.End])
}

func prepareTransformation(ctx context.Context, tCtx base.TransformContext, statement string) ([]statementInfo, error) {
	stmts, err := ParsePg(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	if tCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.New("GetDatabaseMetadataFunc is not set in TransformContext")
	}

	_, metadata, err := tCtx.GetDatabaseMetadataFunc(ctx, tCtx.InstanceID, tCtx.DatabaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database metadata")
	}

	searchPath := metadata.GetSearchPath()
	var dmls []statementInfo

	for i, stmt := range stmts {
		if stmt.Empty() {
			continue
		}

		switch n := stmt.AST.(type) {
		case *ast.VariableSetStmt:
			if strings.EqualFold(n.Name, "search_path") && n.Args != nil {
				var newSearchPath []string
				for _, arg := range n.Args.Items {
					if ac, ok := arg.(*ast.A_Const); ok {
						if s, ok := ac.Val.(*ast.String); ok {
							newSearchPath = append(newSearchPath, s.Str)
						}
					}
				}
				if len(newSearchPath) > 0 {
					searchPath = newSearchPath
				}
			}
		case *ast.UpdateStmt:
			table, err := extractTableReferenceFromRangeVar(n.Relation, metadata, searchPath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract table reference from update statement at offset %d", i)
			}
			if table == nil {
				continue
			}
			table.StatementType = StatementTypeUpdate
			nodeLoc := n.Loc
			startPos := ByteOffsetToRunePosition(statement, nodeLoc.Start)
			startPos.Column-- // 0-based column to match existing convention
			endPos := ByteOffsetToRunePosition(statement, nodeLoc.End)
			dmls = append(dmls, statementInfo{
				offset:    i,
				statement: stmt.Text,
				node:      n,
				table:     table,
				startPos:  startPos,
				endPos:    endPos,
				fullSQL:   statement,
			})
		case *ast.DeleteStmt:
			table, err := extractTableReferenceFromRangeVar(n.Relation, metadata, searchPath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract table reference from delete statement at offset %d", i)
			}
			if table == nil {
				continue
			}
			table.StatementType = StatementTypeDelete
			nodeLoc := n.Loc
			startPos := ByteOffsetToRunePosition(statement, nodeLoc.Start)
			startPos.Column--
			endPos := ByteOffsetToRunePosition(statement, nodeLoc.End)
			dmls = append(dmls, statementInfo{
				offset:    i,
				statement: stmt.Text,
				node:      n,
				table:     table,
				startPos:  startPos,
				endPos:    endPos,
				fullSQL:   statement,
			})
		default:
		}
	}

	return dmls, nil
}

func extractTableReferenceFromRangeVar(rv *ast.RangeVar, metadata *model.DatabaseMetadata, searchPath []string) (*TableReference, error) {
	if rv == nil {
		return nil, nil
	}

	table := TableReference{}

	switch {
	case rv.Catalogname != "":
		table.Database = rv.Catalogname
		table.Schema = rv.Schemaname
		table.Table = rv.Relname
	case rv.Schemaname != "":
		table.Schema = rv.Schemaname
		table.Table = rv.Relname
	default:
		// TODO: remove it in the future.
		// Handle the case where the search path is not synchronized with the metadata.
		if len(searchPath) == 0 {
			searchPath = []string{"public"}
		}
		schemaName, _ := metadata.SearchObject(searchPath, rv.Relname)
		if schemaName == "" {
			return nil, errors.Errorf("Table %q not found in metadata with search path %v", rv.Relname, searchPath)
		}
		table.Schema = schemaName
		table.Table = rv.Relname
	}

	if rv.Alias != nil && rv.Alias.Aliasname != "" {
		table.Alias = rv.Alias.Aliasname
	}
	return &table, nil
}
