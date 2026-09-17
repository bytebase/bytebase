package tsql

import (
	"context"
	"fmt"
	"log/slog"
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

func TransformDMLToSelect(_ context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	statementInfoList, err := prepareTransformation(sourceDatabase, statement, tCtx.IsCaseSensitive)
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
	cteClause, err := backupCTEClause(statementInfoList)
	if err != nil {
		return nil, err
	}
	if cteClause != "" {
		if _, err := fmt.Fprintf(&buf, "%s\n", cteClause); err != nil {
			return nil, errors.Wrap(err, "failed to write buffer")
		}
	}
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

// backupCTEClause returns the WITH clause to place ahead of the backup SELECT
// INTO. T-SQL only allows a CTE list at the top of a statement, so a CTE
// referenced inside a UNION branch must be hoisted to the shared clause. The
// hoisted clause becomes the name scope of every branch, so it is only safe
// when each statement in the group carries the same clause text; a branch
// written without that clause could otherwise resolve a base table name to a
// CTE.
func backupCTEClause(statementInfoList []statementInfo) (string, error) {
	var clause string
	for i, item := range statementInfoList {
		withClause := dmlWithClause(item.node)
		var text string
		if withClause != nil {
			if withClause.XmlNamespaces != nil {
				return "", errors.New("prior backup does not support WITH XMLNAMESPACES")
			}
			text = sourceFromLoc(item.statement, withClause.Loc)
		}
		if i == 0 {
			clause = text
			continue
		}
		if text != clause {
			return "", errors.New("prior backup cannot handle statements with different WITH clauses on the same table")
		}
	}
	return clause, nil
}

func dmlWithClause(node ast.Node) *ast.WithClause {
	switch n := node.(type) {
	case *ast.UpdateStmt:
		return n.WithClause
	case *ast.DeleteStmt:
		return n.WithClause
	default:
		return nil
	}
}

// isCTETarget reports whether an unqualified DML target names a CTE declared
// by the statement's own WITH clause. Such a DML writes through the CTE to a
// base table the backup cannot resolve, so the caller rejects it rather than
// letting the migration run without backup data. Name comparison follows the
// instance collation: a case-sensitive collation keeps c and C distinct.
func isCTETarget(ref *ast.TableRef, withClause *ast.WithClause, caseSensitive bool) bool {
	if ref == nil || withClause == nil || withClause.CTEs == nil || ref.Database != "" || ref.Schema != "" {
		return false
	}
	for _, cteNode := range withClause.CTEs.Items {
		cte, ok := cteNode.(*ast.CommonTableExpr)
		if !ok {
			continue
		}
		if cte.Name == ref.Object || (!caseSensitive && strings.EqualFold(cte.Name, ref.Object)) {
			return true
		}
	}
	return false
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

func prepareTransformation(databaseName, statement string, caseSensitive bool) ([]statementInfo, error) {
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
			targetRef     *ast.TableRef
			statementType StatementType
			err           error
		)
		switch n := node.(type) {
		case *ast.UpdateStmt:
			table, targetRef, err = resolveDMLTargetTable(n.Relation, n.FromClause, databaseName)
			statementType = StatementTypeUpdate
		case *ast.DeleteStmt:
			table, targetRef, err = resolveDMLTargetTable(n.Relation, n.FromClause, databaseName)
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
		if strings.HasPrefix(table.Table, "#") {
			slog.Info("prior backup: skipping DML targeting temp table",
				"table", table.Table,
				"statementType", statementType)
			continue
		}
		if isCTETarget(targetRef, dmlWithClause(node), caseSensitive) {
			return nil, errors.Errorf("prior backup does not support DML targeting CTE %q, target the base table directly", table.Table)
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

// resolveDMLTargetTable returns the DML target and the TableRef it was
// resolved from. When the target is an alias declared in the FROM clause, both
// describe the aliased physical table.
func resolveDMLTargetTable(relation ast.TableExpr, fromClause *ast.List, databaseName string) (*TableReference, *ast.TableRef, error) {
	ref, ok := relation.(*ast.TableRef)
	if !ok {
		return nil, nil, errors.Errorf("unsupported DML target table source %T", relation)
	}
	table := tableReferenceFromTableRef(ref, databaseName, defaultSchema)
	if fromClause != nil && table.Database == databaseName && table.Schema == defaultSchema {
		if physical, physicalRef := findPhysicalTableForAlias(fromClause, table); physical != nil {
			return physical, physicalRef, nil
		}
	}
	return table, ref, nil
}

func tableReferenceFromTableRef(ref *ast.TableRef, defaultDatabase, defaultSchema string) *TableReference {
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
	}
}

func findPhysicalTableForAlias(list *ast.List, table *TableReference) (*TableReference, *ast.TableRef) {
	if list == nil || table == nil {
		return nil, nil
	}
	for _, item := range list.Items {
		if result, ref := findPhysicalTableForAliasInNode(item, table); result != nil {
			return result, ref
		}
	}
	return nil, nil
}

func findPhysicalTableForAliasInNode(node ast.Node, table *TableReference) (*TableReference, *ast.TableRef) {
	switch n := node.(type) {
	case *ast.TableRef:
		if n.Alias != "" && n.Alias == table.Table {
			result := tableReferenceFromTableRef(n, table.Database, table.Schema)
			result.Alias = n.Alias
			return result, n
		}
	case *ast.AliasedTableRef:
		if ref, ok := n.Table.(*ast.TableRef); ok && n.Alias == table.Table {
			result := tableReferenceFromTableRef(ref, table.Database, table.Schema)
			result.Alias = n.Alias
			return result, ref
		}
	case *ast.JoinClause:
		if result, ref := findPhysicalTableForAliasInNode(n.Left, table); result != nil {
			return result, ref
		}
		return findPhysicalTableForAliasInNode(n.Right, table)
	default:
	}
	return nil, nil
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
	end := min(loc.End, len(source))
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
	if common.IsNil(node) {
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
