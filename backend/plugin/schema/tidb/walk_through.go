package tidb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/bytebase/omni/tidb/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterWalkThrough(storepb.Engine_TIDB, WalkThrough)
}

// WalkThrough simulates the DDL in asts against the database state d using the
// omni TiDB catalog, then replaces d with the resulting schema so downstream
// advisor rules see the post-DDL metadata. It mirrors the MySQL walk-through:
//
//  1. Create the catalog and select the target database.
//  2. catalog.LoadMetadata installs the existing schema, with stand-ins for
//     objects it cannot fully model.
//  3. catalog.Exec runs the user DDL.
//  4. The first simulation error becomes an *storepb.Advice.
//  5. The updated catalog is converted back to DatabaseMetadata.
//
// The registered signature carries no raw SQL, so the statement text is
// rebuilt by joining each omni AST's original text.
func WalkThrough(d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
	rawSQL, execASTs := tidbBuildRawSQL(asts)
	if strings.TrimSpace(rawSQL) == "" {
		return nil
	}
	dbName := d.GetProto().GetName()

	precheck := precheckTiDBOmniWalkThrough(d, asts)

	// Step 1: create the catalog and select the target database. LoadMetadata
	// creates and selects it; cross-database rename destinations are seeded up
	// front so a rename into an existing database is not read as missing.
	c := catalog.New()
	initSQL := "SET foreign_key_checks = 0;"
	for _, targetDB := range tidbRenameTargetDatabases(d, asts) {
		initSQL += fmt.Sprintf("\nCREATE DATABASE IF NOT EXISTS %s;", tidbQuoteIdentifier(targetDB))
	}
	if _, err := c.Exec(initSQL, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
		return tidbDDLSimulationAdvice("Failed to initialize catalog", err)
	}

	// Step 2: install the existing schema.
	if _, err := c.LoadMetadata(context.Background(), d.GetProto()); err != nil {
		return tidbDDLSimulationAdvice("Failed to load schema", err)
	}

	// Step 3: execute the user DDL.
	results, execErr := c.Exec(rawSQL, &catalog.ExecOptions{ContinueOnError: true})
	if execErr != nil {
		execAdvice := tidbExecErrorAdvice(d, -1, &storepb.Position{Line: 0}, execErr)
		if precheck.beforeOrSame(execAdvice) {
			return precheck.advice
		}
		return execAdvice.advice
	}

	// Step 4: report the first error from the simulation, anchored to the
	// originating statement's position in the submitted SQL (not the rebuilt
	// rawSQL that tidbBuildRawSQL produced).
	var execAdvice tiDBWalkThroughAdvice
	for _, r := range results {
		if r.Error == nil {
			continue
		}
		execAdvice = tidbExecErrorAdvice(d, r.Index, tidbExecStatementPosition(execASTs, r.Index), r.Error)
		break
	}
	if precheck.beforeOrSame(execAdvice) {
		return precheck.advice
	}
	if execAdvice.advice != nil {
		return execAdvice.advice
	}

	// Step 5: convert the updated catalog back to DatabaseMetadata, keeping the
	// database's case sensitivity (TiDB is case-insensitive) so downstream
	// advisors can still find objects by the statement's spelling.
	newProto := tidbCatalogToProto(c, dbName)
	newMetadata := model.NewDatabaseMetadata(newProto, nil, d.GetConfig(), storepb.Engine_TIDB, d.GetIsObjectCaseSensitive())
	d.ReplaceFrom(newMetadata)
	return nil
}

// tidbBuildRawSQL rebuilds the original SQL from the omni ASTs. Each statement
// keeps its own text; they are re-joined with a terminator so catalog.Exec
// re-parses them as a statement list. The returned slice is parallel to that
// list: execASTs[i] is the base.AST that produced the i-th executed statement,
// so a simulation error's index maps back to the submitted statement's
// position.
func tidbBuildRawSQL(asts []base.AST) (string, []base.AST) {
	var b strings.Builder
	execASTs := make([]base.AST, 0, len(asts))
	for _, a := range asts {
		omniAST, ok := a.(*tidbparser.OmniAST)
		if !ok {
			continue
		}
		text := strings.TrimSpace(omniAST.Text)
		if text == "" {
			continue
		}
		b.WriteString(strings.TrimRight(text, ";"))
		b.WriteString(";\n")
		execASTs = append(execASTs, a)
	}
	return b.String(), execASTs
}

// tidbExecStatementPosition maps a catalog result index back to the originating
// statement's position in the submitted SQL.
func tidbExecStatementPosition(execASTs []base.AST, index int) *storepb.Position {
	if index >= 0 && index < len(execASTs) {
		if pos := execASTs[index].ASTStartPosition(); pos != nil {
			return &storepb.Position{Line: pos.Line, Column: pos.Column}
		}
	}
	return &storepb.Position{Line: 0}
}

// tidbRenameTargetDatabases collects the destination databases of RENAME TABLE
// (and ALTER TABLE ... RENAME TO) statements whose source is the current
// database but whose destination is a different, existing one, mirroring the
// MySQL walk-through so a cross-database rename is not treated as missing.
func tidbRenameTargetDatabases(d *model.DatabaseMetadata, asts []base.AST) []string {
	seen := make(map[string]bool)
	var targets []string
	addTarget := func(sourceTable, targetTable *ast.TableRef) {
		if sourceTable == nil || targetTable == nil {
			return
		}
		if sourceTable.Schema != "" && !isCurrentDatabase(d, sourceTable.Schema) {
			return
		}
		if targetTable.Schema == "" || isCurrentDatabase(d, targetTable.Schema) || seen[targetTable.Schema] {
			return
		}
		seen[targetTable.Schema] = true
		targets = append(targets, targetTable.Schema)
	}
	for _, unifiedAST := range asts {
		node, ok := tidbparser.GetOmniNode(unifiedAST)
		if !ok {
			continue
		}
		switch stmt := node.(type) {
		case *ast.RenameTableStmt:
			for _, pair := range stmt.Pairs {
				if pair == nil {
					continue
				}
				addTarget(pair.Old, pair.New)
			}
		case *ast.AlterTableStmt:
			for _, cmd := range stmt.Commands {
				if cmd == nil || cmd.Type != ast.ATRenameTable {
					continue
				}
				addTarget(stmt.Table, cmd.NewTable)
			}
		default:
		}
	}
	return targets
}

func tidbQuoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

func tidbDDLSimulationAdvice(title string, err error) *storepb.Advice {
	return &storepb.Advice{
		Status:        storepb.Advice_ERROR,
		Code:          code.DDLSimulationFailed.Int32(),
		Title:         title,
		Content:       err.Error(),
		StartPosition: &storepb.Position{Line: 0},
	}
}

type tiDBWalkThroughAdvice struct {
	index  int
	advice *storepb.Advice
}

func (a tiDBWalkThroughAdvice) beforeOrSame(b tiDBWalkThroughAdvice) bool {
	if a.advice == nil {
		return false
	}
	if b.advice == nil {
		return true
	}
	if a.index >= 0 && b.index >= 0 {
		if a.index != b.index {
			return a.index < b.index
		}
		if a.advice.Status != b.advice.Status {
			return a.advice.Status > b.advice.Status
		}
		return true
	}
	if a.index >= 0 && b.index < 0 {
		return true
	}
	if a.index < 0 && b.index >= 0 {
		return false
	}
	return true
}

// tidbExecErrorAdvice maps an omni catalog execution error to an advice.
func tidbExecErrorAdvice(d *model.DatabaseMetadata, index int, position *storepb.Position, err error) tiDBWalkThroughAdvice {
	errCode := code.DDLSimulationFailed
	content := err.Error()
	var catErr *catalog.Error
	if errors.As(err, &catErr) {
		errCode = mapTiDBErrorToCode(catErr)
		content = tidbCatalogErrorAdviceContent(d, catErr)
	}
	return tiDBWalkThroughAdvice{
		index: index,
		advice: &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          errCode.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: position,
		},
	}
}

// precheckTiDBOmniWalkThrough reports the first statement that the catalog
// would silently accept but that bytebase still warns on: a cross-database
// reference, a dropped database, or a CREATE TABLE ... AS SELECT.
func precheckTiDBOmniWalkThrough(d *model.DatabaseMetadata, asts []base.AST) tiDBWalkThroughAdvice {
	// created tracks the tables a preceding statement in this batch added to the
	// current database, so a later CREATE TABLE IF NOT EXISTS against one of them
	// is recognized as the no-op it is rather than flagged.
	created := make(map[string]bool)
	for i, unifiedAST := range asts {
		node, ok := tidbparser.GetOmniNode(unifiedAST)
		if !ok {
			continue
		}
		text := ""
		if omniAST, ok := unifiedAST.(*tidbparser.OmniAST); ok {
			text = strings.TrimSpace(omniAST.Text)
		}
		if advice := precheckTiDBOmniNode(d, node, text, created); advice != nil {
			advice.StartPosition = tidbStatementStartPosition(unifiedAST.ASTStartPosition())
			return tiDBWalkThroughAdvice{index: i, advice: advice}
		}
		tidbTrackBatchTables(d, node, created)
	}
	return tiDBWalkThroughAdvice{}
}

// tidbTrackBatchTables records the current-database tables a statement creates
// or drops, so later prechecks in the same batch see the running table set.
func tidbTrackBatchTables(d *model.DatabaseMetadata, node ast.Node, created map[string]bool) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Table != nil && (n.Table.Schema == "" || isCurrentDatabase(d, n.Table.Schema)) {
			created[tidbTableKey(d, n.Table.Name)] = true
		}
	case *ast.DropTableStmt:
		for _, table := range n.Tables {
			if table != nil && (table.Schema == "" || isCurrentDatabase(d, table.Schema)) {
				delete(created, tidbTableKey(d, table.Name))
			}
		}
	default:
	}
}

func tidbTableKey(d *model.DatabaseMetadata, name string) string {
	if !d.GetIsObjectCaseSensitive() {
		return strings.ToLower(name)
	}
	return name
}

func precheckTiDBOmniNode(d *model.DatabaseMetadata, node ast.Node, text string, created map[string]bool) *storepb.Advice {
	switch n := node.(type) {
	case *ast.CreateDatabaseStmt:
		if n.Name != "" && !isCurrentDatabase(d, n.Name) {
			return notCurrentTiDBDatabaseAdvice(d, n.Name)
		}
	case *ast.AlterDatabaseStmt:
		if n.Name != "" && !isCurrentDatabase(d, n.Name) {
			return notCurrentTiDBDatabaseAdvice(d, n.Name)
		}
	case *ast.DropDatabaseStmt:
		if n.Name != "" && !isCurrentDatabase(d, n.Name) {
			return notCurrentTiDBDatabaseAdvice(d, n.Name)
		}
		if n.Name != "" {
			content := fmt.Sprintf("Database `%s` is deleted", n.Name)
			return &storepb.Advice{
				Status:        storepb.Advice_WARNING,
				Code:          code.DatabaseIsDeleted.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
		}
	case *ast.CreateTableStmt:
		if advice := precheckTiDBTableRefDatabase(d, n.Table); advice != nil {
			return advice
		}
		if n.IfNotExists && tidbTableExistsInBatch(d, created, n.Table) {
			return nil
		}
		if n.Select != nil {
			content := fmt.Sprintf("CREATE TABLE AS statement is used in \"%s\"", text)
			return &storepb.Advice{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementCreateTableAs.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
		}
	case *ast.AlterTableStmt:
		return precheckTiDBTableRefDatabase(d, n.Table)
	case *ast.DropTableStmt:
		for _, table := range n.Tables {
			if advice := precheckTiDBTableRefDatabase(d, table); advice != nil {
				return advice
			}
		}
	case *ast.CreateIndexStmt:
		return precheckTiDBTableRefDatabase(d, n.Table)
	case *ast.DropIndexStmt:
		return precheckTiDBTableRefDatabase(d, n.Table)
	case *ast.CreateViewStmt:
		return precheckTiDBTableRefDatabase(d, n.Name)
	case *ast.RenameTableStmt:
		for _, pair := range n.Pairs {
			if pair == nil {
				continue
			}
			if advice := precheckTiDBTableRefDatabase(d, pair.Old); advice != nil {
				return advice
			}
		}
	default:
	}
	return nil
}

func tidbStatementStartPosition(pos *storepb.Position) *storepb.Position {
	if pos == nil {
		return &storepb.Position{Line: 0}
	}
	return &storepb.Position{Line: pos.Line, Column: pos.Column}
}

func precheckTiDBTableRefDatabase(d *model.DatabaseMetadata, table *ast.TableRef) *storepb.Advice {
	if table == nil || table.Schema == "" || isCurrentDatabase(d, table.Schema) {
		return nil
	}
	return notCurrentTiDBDatabaseAdvice(d, table.Schema)
}

// tidbTableExistsInBatch reports whether a table exists either in the loaded
// metadata or by a preceding statement in the same batch.
func tidbTableExistsInBatch(d *model.DatabaseMetadata, created map[string]bool, table *ast.TableRef) bool {
	if tidbOmniTableExistsInCurrentDatabase(d, table) {
		return true
	}
	if table == nil || (table.Schema != "" && !isCurrentDatabase(d, table.Schema)) {
		return false
	}
	return created[tidbTableKey(d, table.Name)]
}

func tidbOmniTableExistsInCurrentDatabase(d *model.DatabaseMetadata, table *ast.TableRef) bool {
	if table == nil || (table.Schema != "" && !isCurrentDatabase(d, table.Schema)) {
		return false
	}
	s := d.GetSchemaMetadata("")
	if s == nil {
		return false
	}
	return s.GetTable(table.Name) != nil
}

func notCurrentTiDBDatabaseAdvice(d *model.DatabaseMetadata, databaseName string) *storepb.Advice {
	content := fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, d.DatabaseName())
	return &storepb.Advice{
		Status:        storepb.Advice_WARNING,
		Code:          code.NotCurrentDatabase.Int32(),
		Title:         content,
		Content:       content,
		StartPosition: &storepb.Position{Line: 0},
	}
}

// mapTiDBErrorToCode converts an omni TiDB catalog error to a bytebase advisor
// code. The catalog uses MySQL-compatible error numbers, so this mirrors the
// MySQL walk-through's mapping.
func mapTiDBErrorToCode(catErr *catalog.Error) code.Code {
	switch catErr.Code {
	case catalog.ErrDupDatabase:
		return code.DDLSimulationFailed
	case catalog.ErrUnknownDatabase:
		return code.NotCurrentDatabase
	case catalog.ErrDupTable:
		return code.TableExists
	case catalog.ErrUnknownTable, catalog.ErrNoSuchTable:
		return code.TableNotExists
	case catalog.ErrDupColumn:
		return code.ColumnExists
	case catalog.ErrNoSuchColumn, catalog.ErrKeyColumnDoesNotExist:
		return code.ColumnNotExists
	case catalog.ErrInvalidDefault, catalog.ErrBlobCantHaveDefault:
		return code.InvalidColumnDefault
	case catalog.ErrDupKeyName, catalog.ErrDupIndex, catalog.ErrDupEntry:
		return code.IndexExists
	case catalog.ErrMultiplePriKey:
		return code.PrimaryKeyExists
	case catalog.ErrCantDropKey:
		// 1091 is shared: DROP COLUMN of a missing column ("Can't DROP ...")
		// is a missing column; DROP INDEX of a missing index is a missing
		// index.
		if tidbIsDropColumnMessage(catErr.Message) {
			return code.ColumnNotExists
		}
		return code.IndexNotExists
	case catalog.ErrIncorrectTableDefinition:
		return code.AutoIncrementExists
	case catalog.ErrInvalidOnUpdate:
		return code.OnUpdateColumnNotDatetimeOrTimestamp
	default:
		return code.DDLSimulationFailed
	}
}

// tidbCatalogErrorAdviceContent renders a catalog error as user-facing advice,
// mirroring the MySQL walk-through's phrasing.
func tidbCatalogErrorAdviceContent(d *model.DatabaseMetadata, catErr *catalog.Error) string {
	name := tidbCatalogErrorName(catErr.Message)
	switch catErr.Code {
	case catalog.ErrUnknownDatabase:
		return fmt.Sprintf("Database `%s` is not the current database `%s`", name, d.DatabaseName())
	case catalog.ErrDupTable:
		return fmt.Sprintf("Table `%s` already exists", tidbUnqualifiedName(name))
	case catalog.ErrUnknownTable, catalog.ErrNoSuchTable:
		return fmt.Sprintf("Table `%s` does not exist", tidbUnqualifiedName(name))
	case catalog.ErrDupColumn:
		return fmt.Sprintf("Column `%s` already exists", tidbUnqualifiedName(name))
	case catalog.ErrNoSuchColumn:
		return fmt.Sprintf("Column `%s` does not exist", tidbUnqualifiedName(name))
	case catalog.ErrKeyColumnDoesNotExist:
		return fmt.Sprintf("Column `%s` does not exist", tidbKeyColumnName(catErr.Message))
	case catalog.ErrInvalidDefault:
		return fmt.Sprintf("Invalid default value for column `%s`", tidbUnqualifiedName(name))
	case catalog.ErrBlobCantHaveDefault:
		return fmt.Sprintf("BLOB, TEXT or JSON column `%s` can't have a default value", tidbUnqualifiedName(name))
	case catalog.ErrDupKeyName, catalog.ErrDupIndex, catalog.ErrDupEntry:
		return fmt.Sprintf("Index `%s` already exists", tidbUnqualifiedName(name))
	case catalog.ErrCantDropKey:
		if tidbIsDropColumnMessage(catErr.Message) {
			return fmt.Sprintf("Column `%s` does not exist", tidbCantDropName(catErr.Message))
		}
		return fmt.Sprintf("Index `%s` does not exist", tidbDroppedIndexName(catErr.Message))
	case catalog.ErrMultiplePriKey:
		return "Multiple primary key defined"
	case catalog.ErrIncorrectTableDefinition:
		return "There can be only one auto column and it must be defined as a key"
	case catalog.ErrInvalidOnUpdate:
		return fmt.Sprintf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", tidbUnqualifiedName(name))
	default:
		return catErr.Message
	}
}

func tidbCatalogErrorName(message string) string {
	start := strings.Index(message, "'")
	if start < 0 {
		return message
	}
	end := strings.Index(message[start+1:], "'")
	if end < 0 {
		return message[start+1:]
	}
	return message[start+1 : start+1+end]
}

func tidbUnqualifiedName(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// The ErrCantDropKey (1091) and ErrKeyColumnDoesNotExist (1072) messages carry
// the object name outside single quotes (and contain apostrophes in "Can't" and
// "doesn't"), so they need their own extraction rather than the quoted-token one.

// tidbIsDropColumnMessage reports whether a 1091 error is a DROP COLUMN of a
// missing column ("Can't DROP '<col>'; ...") rather than a DROP INDEX of a
// missing index ("index <name> doesn't exist").
func tidbIsDropColumnMessage(message string) bool {
	return strings.HasPrefix(message, "Can't DROP")
}

// tidbCantDropName extracts <col> from "Can't DROP '<col>'; check that ...".
func tidbCantDropName(message string) string {
	const prefix = "Can't DROP '"
	rest, ok := strings.CutPrefix(message, prefix)
	if !ok {
		return message
	}
	if idx := strings.IndexByte(rest, '\''); idx >= 0 {
		return rest[:idx]
	}
	return rest
}

// tidbDroppedIndexName extracts <name> from "index <name> doesn't exist".
func tidbDroppedIndexName(message string) string {
	rest := strings.TrimPrefix(message, "index ")
	return strings.TrimSuffix(rest, " doesn't exist")
}

// tidbKeyColumnName extracts <col> from "column does not exist: <col>".
func tidbKeyColumnName(message string) string {
	return strings.TrimPrefix(message, "column does not exist: ")
}

func isCurrentDatabase(d *model.DatabaseMetadata, database string) bool {
	if !d.GetIsObjectCaseSensitive() {
		return strings.EqualFold(d.DatabaseName(), database)
	}
	return d.DatabaseName() == database
}
