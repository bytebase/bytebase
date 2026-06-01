package plsql

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	oracleast "github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	maxTableNameLengthAfter12_2  = 128
	maxTableNameLengthBefore12_2 = 30
)

var errNoBackupableDML = errors.New("no parse results")

func init() {
	base.RegisterTransformDMLToSelect(store.Engine_ORACLE, TransformDMLToSelect)
}

type StatementType int

const (
	StatementTypeUnknown StatementType = iota
	StatementTypeUpdate
	StatementTypeInsert
	StatementTypeDelete
)

type TableReference struct {
	Database      string
	HasSchema     bool
	Schema        string
	Table         string
	Alias         string
	StatementType StatementType
}

type statementInfo struct {
	offset        int
	statement     string
	node          oracleast.StmtNode
	table         *TableReference
	startPosition *store.Position
	endPosition   *store.Position
	fullSQL       string
}

// TransformDMLToSelect transforms DML statement to SELECT statement.
// For Oracle, we only consider the managed on schema mode.
func TransformDMLToSelect(_ context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(sourceDatabase, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	return generateSQL(tCtx, statementInfoList, targetDatabase, tablePrefix)
}

func generateSQL(ctx base.TransformContext, statementInfoList []statementInfo, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s", item.table.Schema, item.table.Table)
		groupByTable[key] = append(groupByTable[key], item)
	}

	// Check if the statement type is the same for all statements in the group.
	for key, list := range groupByTable {
		statementType := StatementTypeUnknown
		for _, item := range list {
			if statementType == StatementTypeUnknown {
				statementType = item.table.StatementType
			}
			if statementType != item.table.StatementType {
				return nil, errors.Errorf("prior backup cannot handle statements with different types on the same table: %s", key)
			}
		}
	}

	var result []base.BackupStatement
	for key, list := range groupByTable {
		backupStatement, err := generateSQLForTable(ctx, list, targetDatabase, tablePrefix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate SQL for table: %s", key)
		}
		result = append(result, *backupStatement)
	}

	slices.SortFunc(result, func(i, j base.BackupStatement) int {
		if i.StartPosition.Line != j.StartPosition.Line {
			if i.StartPosition.Line < j.StartPosition.Line {
				return -1
			}
			return 1
		}
		if i.StartPosition.Column != j.StartPosition.Column {
			if i.StartPosition.Column < j.StartPosition.Column {
				return -1
			}
			return 1
		}
		if i.SourceTableName < j.SourceTableName {
			return -1
		}
		if i.SourceTableName > j.SourceTableName {
			return 1
		}
		return 0
	})

	return result, nil
}

func generateSQLForTable(ctx base.TransformContext, statementInfoList []statementInfo, targetDatabase string, tablePrefix string) (*base.BackupStatement, error) {
	table := statementInfoList[0].table

	version, ok := ctx.Version.(*Version)
	if !ok {
		version = &Version{
			First:  11,
			Second: 0,
		}
	}

	targetTable := fmt.Sprintf("%s_%s_%s", tablePrefix, table.Table, table.Schema)
	if version.GTE(&Version{First: 12, Second: 2}) {
		targetTable, _ = common.TruncateString(targetTable, maxTableNameLengthAfter12_2)
	} else {
		targetTable, _ = common.TruncateString(targetTable, maxTableNameLengthBefore12_2)
	}

	var buf strings.Builder
	if _, err := fmt.Fprintf(&buf, `CREATE TABLE "%s"."%s" AS`+"\n", targetDatabase, targetTable); err != nil {
		return nil, errors.Wrap(err, "failed to write to buffer")
	}
	for i, info := range statementInfoList {
		if i != 0 {
			if _, err := buf.WriteString("\n  UNION\n"); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		}
		t := info.table
		if t.Alias != "" {
			if _, err := fmt.Fprintf(&buf, `  SELECT "%s".* FROM `, t.Alias); err != nil {
				return nil, errors.Wrap(err, "failed to write to buffer")
			}
		} else {
			if t.HasSchema {
				if _, err := fmt.Fprintf(&buf, `  SELECT "%s"."%s".* FROM `, t.Schema, t.Table); err != nil {
					return nil, errors.Wrap(err, "failed to write to buffer")
				}
			} else {
				if _, err := fmt.Fprintf(&buf, `  SELECT "%s".* FROM `, t.Table); err != nil {
					return nil, errors.Wrap(err, "failed to write to buffer")
				}
			}
		}
		if err := writeSuffixSelectClause(&buf, info.node, info.fullSQL); err != nil {
			return nil, errors.Wrap(err, "failed to write suffix select clause")
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
		StartPosition:   zeroBasedColumnPosition(statementInfoList[0].startPosition),
		EndPosition:     zeroBasedColumnPosition(statementInfoList[len(statementInfoList)-1].endPosition),
	}, nil
}

func prepareTransformation(databaseName, statement string) ([]statementInfo, error) {
	statements, err := SplitSQL(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split PLSQL")
	}

	var result []statementInfo
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for i, stmt := range statements {
		if stmt.Empty {
			continue
		}
		list, err := ParsePLSQLOmni(stmt.Text)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse PLSQL")
		}
		for _, item := range list.Items {
			raw, ok := item.(*oracleast.RawStmt)
			if !ok || raw.Stmt == nil {
				continue
			}
			info := extractOmniDML(databaseName, raw.Stmt, stmt.Text)
			if info == nil {
				continue
			}
			info.offset = i
			if stmt.Range != nil {
				info.startPosition = positionMapper.Position(int(stmt.Range.Start) + raw.Loc.Start)
				info.endPosition = positionMapper.Position(int(stmt.Range.Start) + max(raw.Loc.End-1, raw.Loc.Start))
			} else {
				info.startPosition = stmt.Start
				info.endPosition = stmt.End
			}
			info.fullSQL = stmt.Text
			result = append(result, *info)
		}
	}

	if len(result) == 0 {
		return nil, errNoBackupableDML
	}
	return result, nil
}

func extractOmniDML(databaseName string, node oracleast.StmtNode, fullSQL string) *statementInfo {
	switch n := node.(type) {
	case *oracleast.DeleteStmt:
		table := omniDMLTableReference(databaseName, n.Table, n.Alias, StatementTypeDelete)
		if table == nil {
			return nil
		}
		return &statementInfo{
			statement: extractOmniStatementText(fullSQL, n.Loc),
			node:      n,
			table:     table,
		}
	case *oracleast.UpdateStmt:
		table := omniDMLTableReference(databaseName, n.Table, n.Alias, StatementTypeUpdate)
		if table == nil {
			return nil
		}
		return &statementInfo{
			statement: extractOmniStatementText(fullSQL, n.Loc),
			node:      n,
			table:     table,
		}
	default:
		return nil
	}
}

func omniDMLTableReference(databaseName string, name *oracleast.ObjectName, alias *oracleast.Alias, statementType StatementType) *TableReference {
	if name == nil || name.Name == "" {
		return nil
	}
	schema := name.Schema
	hasSchema := schema != ""
	if schema == "" {
		schema = databaseName
	}
	table := &TableReference{
		Database:      schema,
		HasSchema:     hasSchema,
		Schema:        schema,
		Table:         name.Name,
		StatementType: statementType,
	}
	if alias != nil {
		table.Alias = alias.Name
	}
	return table
}

func writeSuffixSelectClause(buf *strings.Builder, node oracleast.StmtNode, fullSQL string) error {
	suffix := ""
	switch n := node.(type) {
	case *oracleast.UpdateStmt:
		suffix = oracleDMLTargetText(fullSQL, n.Table, n.PartitionExt, n.Alias)
		if n.WhereClause != nil && n.SetClauses != nil && n.SetClauses.Len() > 0 {
			if setClause, ok := n.SetClauses.Items[n.SetClauses.Len()-1].(*oracleast.SetClause); ok {
				whereStart := setClause.Loc.End
				if fromText, fromEnd, ok := oracleListText(fullSQL, n.FromClause); ok {
					suffix += ", " + fromText
					whereStart = fromEnd
				}
				suffix = joinOracleSuffix(suffix, oracleWhereSuffix(fullSQL, whereStart, n.WhereClause))
			}
		}
	case *oracleast.DeleteStmt:
		targetEnd := n.Table.Loc.End
		if n.PartitionExt != nil {
			targetEnd = n.PartitionExt.Loc.End
		}
		if n.Alias != nil {
			targetEnd = n.Alias.Loc.End
		}
		suffix = joinOracleSuffix(oracleDMLTargetText(fullSQL, n.Table, n.PartitionExt, n.Alias), oracleWhereSuffix(fullSQL, targetEnd, n.WhereClause))
	default:
	}
	_, err := buf.WriteString(suffix)
	return err
}

func oracleDMLTargetText(sql string, name *oracleast.ObjectName, partitionExt *oracleast.PartitionExtClause, alias *oracleast.Alias) string {
	if name == nil {
		return ""
	}
	end := name.Loc.End
	if partitionExt != nil {
		end = partitionExt.Loc.End
	}
	if alias != nil {
		end = alias.Loc.End
	}
	return strings.TrimSpace(oracleLocText(sql, name.Loc.Start, end))
}

func oracleDMLTrailingText(sql string, start, end int) string {
	text := strings.TrimSpace(oracleLocText(sql, start, end))
	return strings.TrimSpace(strings.TrimSuffix(text, ";"))
}

func oracleListText(sql string, list *oracleast.List) (string, int, bool) {
	if list == nil || list.Len() == 0 {
		return "", 0, false
	}
	start, ok := oracleNodeLoc(list.Items[0])
	if !ok {
		return "", 0, false
	}
	end, ok := oracleNodeLoc(list.Items[list.Len()-1])
	if !ok {
		return "", 0, false
	}
	text := strings.TrimSpace(oracleLocText(sql, start.Start, end.End))
	if text == "" {
		return "", 0, false
	}
	return text, end.End, true
}

func oracleWhereSuffix(sql string, start int, where oracleast.ExprNode) string {
	loc, ok := oracleNodeLoc(where)
	if !ok {
		return ""
	}
	return oracleDMLTrailingText(sql, start, loc.End)
}

func oracleNodeLoc(node oracleast.Node) (oracleast.Loc, bool) {
	if node == nil {
		return oracleast.Loc{}, false
	}
	value := reflect.ValueOf(node)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return oracleast.Loc{}, false
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return oracleast.Loc{}, false
	}
	field := elem.FieldByName("Loc")
	if !field.IsValid() || field.Type() != reflect.TypeOf(oracleast.Loc{}) {
		return oracleast.Loc{}, false
	}
	loc, ok := field.Interface().(oracleast.Loc)
	if !ok || loc.End <= loc.Start {
		return oracleast.Loc{}, false
	}
	return loc, true
}

func oracleLocText(sql string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end <= 0 || end > len(sql) {
		end = len(sql)
	}
	if start >= end || start >= len(sql) {
		return ""
	}
	return sql[start:end]
}

func joinOracleSuffix(parts ...string) string {
	var nonEmpty []string
	for _, part := range parts {
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, " ")
}

func extractOmniStatementText(sql string, loc oracleast.Loc) string {
	return strings.TrimSpace(strings.TrimSuffix(oracleLocText(sql, loc.Start, loc.End), ";"))
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
