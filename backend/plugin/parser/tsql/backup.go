package tsql

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_MSSQL, TransformDMLToSelect)
}

const (
	// The default schema is 'dbo' for MSSQL.
	// TODO(zp): We should support default schema in the future.
	defaultSchema      = "dbo"
	maxTableNameLength = 128
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

type statementInfo struct {
	statement     string
	node          ast.Node
	table         *TableReference
	startPosition *storepb.Position
	endPosition   *storepb.Position
}

func TransformDMLToSelect(_ context.Context, _ base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(sourceDatabase, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare transformation")
	}

	return generateSQL(statementInfoList, targetDatabase, tablePrefix)
}

func generateSQL(statementInfoList []statementInfo, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	groupByTable := make(map[string][]statementInfo)
	for _, item := range statementInfoList {
		key := fmt.Sprintf("%s.%s.%s", item.table.Database, item.table.Schema, item.table.Table)
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
				return nil, errors.Errorf("prior backup cannot handle mixed DMLs on the same table %s", key)
			}
		}
	}

	var result []base.BackupStatement
	for key, list := range groupByTable {
		backupStatement, err := generateSQLForTable(list, targetDatabase, tablePrefix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate SQL for table %s", key)
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

func generateSQLForTable(statementInfoList []statementInfo, targetDatabase string, tablePrefix string) (*base.BackupStatement, error) {
	table := statementInfoList[0].table

	targetTable := fmt.Sprintf("%s_%s_%s", tablePrefix, table.Table, table.Database)
	targetTable, _ = common.TruncateString(targetTable, maxTableNameLength)
	var buf strings.Builder
	if _, err := fmt.Fprintf(&buf, "SELECT * INTO [%s].[%s].[%s] FROM (\n", targetDatabase, defaultSchema, targetTable); err != nil {
		return nil, errors.Wrap(err, "failed to write buffer")
	}
	for i, item := range statementInfoList {
		if i > 0 {
			if _, err := buf.WriteString("\n  UNION\n"); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		}
		topClause, fromClause, err := extractSuffixSelectStatement(item.node, item.statement)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract suffix select statement")
		}
		if len(item.table.Alias) == 0 {
			if _, err := fmt.Fprintf(&buf, "  SELECT [%s].[%s].[%s].* ", item.table.Database, item.table.Schema, item.table.Table); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		} else {
			if _, err := fmt.Fprintf(&buf, "  SELECT [%s].* ", item.table.Alias); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		}
		if len(topClause) > 0 {
			if _, err := buf.WriteString(topClause); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
			if _, err := buf.WriteString(" "); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		}
		if len(fromClause) > 0 {
			if _, err := buf.WriteString(fromClause); err != nil {
				return nil, errors.Wrap(err, "failed to write buffer")
			}
		}
	}
	if _, err := buf.WriteString(") AS backup_table;"); err != nil {
		return nil, errors.Wrap(err, "failed to write buffer")
	}
	return &base.BackupStatement{
		Statement:       buf.String(),
		SourceSchema:    table.Schema,
		SourceTableName: table.Table,
		TargetTableName: targetTable,
		StartPosition:   statementInfoList[0].startPosition,
		EndPosition:     statementInfoList[len(statementInfoList)-1].endPosition,
	}, nil
}

func extractSuffixSelectStatement(node ast.Node, source string) (string, string, error) {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		if _, ok := n.WhereClause.(*ast.CurrentOfExpr); ok {
			return "", "", errors.New("UPDATE statement with CURSOR clause is not supported")
		}
		fromSource, searchStart := dmlFromSource(source, n.Relation, n.FromClause, dmlNodeLoc(n), n.WhereClause, n.OptionClause)
		return sourceFromLoc(source, dmlNodeLoc(n.Top)), buildDMLFromClause(source, fromSource, searchStart, dmlNodeLoc(n), n.WhereClause, n.OptionClause), nil
	case *ast.DeleteStmt:
		if _, ok := n.WhereClause.(*ast.CurrentOfExpr); ok {
			return "", "", errors.New("DELETE statement with CURSOR clause is not supported")
		}
		fromSource, searchStart := dmlFromSource(source, n.Relation, n.FromClause, dmlNodeLoc(n), n.WhereClause, n.OptionClause)
		return sourceFromLoc(source, dmlNodeLoc(n.Top)), buildDMLFromClause(source, fromSource, searchStart, dmlNodeLoc(n), n.WhereClause, n.OptionClause), nil
	default:
		return "", "", nil
	}
}

func prepareTransformation(databaseName, statement string) ([]statementInfo, error) {
	parsedStatements, err := parseTSQLStatements(statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}

	var dmls []statementInfo
	for _, parsedStatement := range parsedStatements {
		node, ok := GetOmniNode(parsedStatement.AST)
		if !ok || node == nil {
			continue
		}
		var (
			table         *TableReference
			statementType StatementType
			err           error
		)
		switch n := node.(type) {
		case *ast.UpdateStmt:
			table, err = resolveDMLTargetTable(n.Relation, n.FromClause, databaseName)
			statementType = StatementTypeUpdate
		case *ast.DeleteStmt:
			table, err = resolveDMLTargetTable(n.Relation, n.FromClause, databaseName)
			statementType = StatementTypeDelete
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
		if table == nil {
			return nil, errors.Errorf("failed to resolve DML target table")
		}
		table.StatementType = statementType
		loc := dmlNodeLoc(node)
		dmls = append(dmls, statementInfo{
			statement:     parsedStatement.Text,
			node:          node,
			table:         table,
			startPosition: positionFromByteOffset(parsedStatement.Start, parsedStatement.Text, loc.Start),
			endPosition:   positionFromByteOffset(parsedStatement.Start, parsedStatement.Text, dmlEndOffset(parsedStatement.Text, loc)),
		})
	}

	return dmls, nil
}

func resolveDMLTargetTable(relation ast.TableExpr, fromClause *ast.List, databaseName string) (*TableReference, error) {
	table, err := tableReferenceFromTableExpr(relation, databaseName, defaultSchema)
	if err != nil || table == nil {
		return table, err
	}
	if fromClause != nil && table.Database == databaseName && table.Schema == defaultSchema {
		if physical := findPhysicalTableForAlias(fromClause, table); physical != nil {
			return physical, nil
		}
	}
	return table, nil
}

func tableReferenceFromTableExpr(expr ast.TableExpr, defaultDatabase, defaultSchema string) (*TableReference, error) {
	ref, ok := expr.(*ast.TableRef)
	if !ok {
		return nil, errors.Errorf("unsupported DML target table source %T", expr)
	}
	schemaName := defaultSchema
	if ref.Schema != "" {
		schemaName = ref.Schema
	}
	databaseName := defaultDatabase
	if ref.Database != "" {
		databaseName = ref.Database
	}
	return &TableReference{
		Database: databaseName,
		Schema:   schemaName,
		Table:    ref.Object,
		Alias:    ref.Alias,
	}, nil
}

func findPhysicalTableForAlias(list *ast.List, table *TableReference) *TableReference {
	if list == nil || table == nil {
		return nil
	}
	for _, item := range list.Items {
		if result := findPhysicalTableForAliasInNode(item, table); result != nil {
			return result
		}
	}
	return nil
}

func findPhysicalTableForAliasInNode(node ast.Node, table *TableReference) *TableReference {
	switch n := node.(type) {
	case *ast.TableRef:
		if n.Alias != "" && n.Alias == table.Table {
			ref, err := tableReferenceFromTableExpr(n, table.Database, table.Schema)
			if err != nil {
				return nil
			}
			ref.Alias = n.Alias
			return ref
		}
	case *ast.AliasedTableRef:
		if ref, ok := n.Table.(*ast.TableRef); ok && n.Alias == table.Table {
			result, err := tableReferenceFromTableExpr(ref, table.Database, table.Schema)
			if err != nil {
				return nil
			}
			result.Alias = n.Alias
			return result
		}
	case *ast.JoinClause:
		if result := findPhysicalTableForAliasInNode(n.Left, table); result != nil {
			return result
		}
		return findPhysicalTableForAliasInNode(n.Right, table)
	default:
	}
	return nil
}

func dmlFromSource(source string, relation ast.TableExpr, fromClause *ast.List, stmtLoc ast.Loc, where ast.ExprNode, option *ast.List) (string, int) {
	if fromClause != nil && len(fromClause.Items) > 0 {
		loc := listLoc(fromClause)
		end := dmlTailStart(source, loc.End, stmtLoc, where, option)
		if end < 0 {
			end = trimDMLStatementEnd(source, stmtLoc.End)
		}
		return strings.TrimSpace(source[loc.Start:end]), end
	}
	loc := dmlNodeLoc(relation)
	return sourceFromLoc(source, loc), loc.End
}

func buildDMLFromClause(source, fromSource string, searchStart int, stmtLoc ast.Loc, where ast.ExprNode, option *ast.List) string {
	if fromSource == "" {
		return ""
	}
	tail := dmlTrailingClause(source, searchStart, stmtLoc, where, option)
	if tail == "" {
		return "FROM " + fromSource
	}
	return "FROM " + fromSource + " " + tail
}

func dmlTrailingClause(source string, searchStart int, stmtLoc ast.Loc, where ast.ExprNode, option *ast.List) string {
	end := trimDMLStatementEnd(source, stmtLoc.End)
	start := dmlTailStart(source, searchStart, stmtLoc, where, option)
	if start < 0 || start >= end {
		return ""
	}
	return strings.TrimSpace(source[start:end])
}

func dmlTailStart(source string, searchStart int, _ ast.Loc, where ast.ExprNode, option *ast.List) int {
	if where != nil {
		if start := findKeywordBefore(source, "WHERE", dmlNodeLoc(where).Start, searchStart); start >= 0 {
			return start
		}
	}
	if option != nil {
		return findKeywordAfter(source, "OPTION", searchStart, len(source))
	}
	return -1
}

func trimDMLStatementEnd(source string, end int) int {
	if end > len(source) {
		end = len(source)
	}
	for end > 0 {
		switch source[end-1] {
		case ';', ' ', '\t', '\r', '\n':
			end--
		default:
			return end
		}
	}
	return end
}

func findKeywordBefore(source, keyword string, before int, after int) int {
	if before > len(source) {
		before = len(source)
	}
	if after < 0 {
		after = 0
	}
	return findKeywordOutsideSQL(source, keyword, after, before, true)
}

func findKeywordAfter(source, keyword string, start int, end int) int {
	if start < 0 {
		start = 0
	}
	if end > len(source) {
		end = len(source)
	}
	return findKeywordOutsideSQL(source, keyword, start, end, false)
}

func findKeywordOutsideSQL(source, keyword string, start, end int, last bool) int {
	if start >= end {
		return -1
	}
	result := -1
	for i := start; i < end; {
		switch source[i] {
		case '\'':
			i = skipQuotedSQL(source, i, end, '\'')
		case '"':
			i = skipQuotedSQL(source, i, end, '"')
		case '[':
			i = skipQuotedSQL(source, i, end, ']')
		case '-':
			if i+1 < end && source[i+1] == '-' {
				i = skipLineComment(source, i+2, end)
				continue
			}
			if keywordMatchAt(source, keyword, i, start, end) {
				if !last {
					return i
				}
				result = i
				i += len(keyword)
				continue
			}
			i++
		case '/':
			if i+1 < end && source[i+1] == '*' {
				i = skipBlockComment(source, i+2, end)
				continue
			}
			if keywordMatchAt(source, keyword, i, start, end) {
				if !last {
					return i
				}
				result = i
				i += len(keyword)
				continue
			}
			i++
		default:
			if keywordMatchAt(source, keyword, i, start, end) {
				if !last {
					return i
				}
				result = i
				i += len(keyword)
				continue
			}
			i++
		}
	}
	return result
}

func keywordMatchAt(source, keyword string, offset, start, end int) bool {
	if offset+len(keyword) > end || !strings.EqualFold(source[offset:offset+len(keyword)], keyword) {
		return false
	}
	if offset > start && isSQLIdentifierChar(source[offset-1]) {
		return false
	}
	if offset+len(keyword) < end && isSQLIdentifierChar(source[offset+len(keyword)]) {
		return false
	}
	return true
}

func isSQLIdentifierChar(ch byte) bool {
	return ch == '_' || ch == '@' || ch == '#' || ch == '$' ||
		(ch >= '0' && ch <= '9') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z')
}

func skipQuotedSQL(source string, start, end int, closeCh byte) int {
	i := start + 1
	for i < end {
		if source[i] == closeCh {
			if i+1 < end && source[i+1] == closeCh {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return end
}

func skipLineComment(source string, start, end int) int {
	for start < end && source[start] != '\n' {
		start++
	}
	return start
}

func skipBlockComment(source string, start, end int) int {
	depth := 1
	for start+1 < end {
		switch {
		case source[start] == '/' && source[start+1] == '*':
			depth++
			start += 2
		case source[start] == '*' && source[start+1] == '/':
			depth--
			start += 2
			if depth == 0 {
				return start
			}
		default:
			start++
		}
	}
	return end
}

func sourceFromLoc(source string, loc ast.Loc) string {
	if loc.Start < 0 || loc.End < 0 || loc.Start >= len(source) {
		return ""
	}
	end := loc.End
	if end > len(source) {
		end = len(source)
	}
	return strings.TrimSpace(source[loc.Start:end])
}

func positionFromByteOffset(start *storepb.Position, source string, offset int) *storepb.Position {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(source) && len(source) > 0 {
		offset = len(source) - 1
	}
	line, column := base.CalculateLineAndColumn(source, offset)
	startLine := int32(1)
	if start != nil {
		startLine = start.Line
	}
	return &storepb.Position{
		Line:   startLine + int32(line),
		Column: int32(column),
	}
}

func dmlEndOffset(source string, loc ast.Loc) int {
	if loc.End >= 0 && loc.End < len(source) && source[loc.End] == ';' {
		return loc.End
	}
	return loc.End - 1
}

func listLoc(list *ast.List) ast.Loc {
	if list == nil || len(list.Items) == 0 {
		return ast.NoLoc()
	}
	return ast.Loc{
		Start: dmlNodeLoc(list.Items[0]).Start,
		End:   dmlNodeLoc(list.Items[len(list.Items)-1]).End,
	}
}

func dmlNodeLoc(node ast.Node) ast.Loc {
	if node == nil {
		return ast.NoLoc()
	}
	v := reflect.ValueOf(node)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return ast.NoLoc()
	}
	switch n := node.(type) {
	case *ast.UpdateStmt:
		return n.Loc
	case *ast.DeleteStmt:
		return n.Loc
	case *ast.TopClause:
		return n.Loc
	case *ast.TableRef:
		return n.Loc
	case *ast.TableVarRef:
		return n.Loc
	case *ast.TableVarMethodCallRef:
		return n.Loc
	case *ast.AliasedTableRef:
		return n.Loc
	case *ast.JoinClause:
		loc := n.Loc
		leftLoc := dmlNodeLoc(n.Left)
		if leftLoc.Start >= 0 && (loc.Start < 0 || leftLoc.Start < loc.Start) {
			loc.Start = leftLoc.Start
		}
		return loc
	case *ast.SetExpr:
		return n.Loc
	default:
		loc := omniNodeLoc(node)
		if loc.Start >= 0 {
			return loc
		}
		return ast.NoLoc()
	}
}
