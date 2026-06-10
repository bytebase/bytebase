package mysql

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterWalkThroughWithContext(storepb.Engine_MYSQL, WalkThroughOmni)
	schema.RegisterWalkThroughWithContext(storepb.Engine_MARIADB, WalkThroughOmni)
	schema.RegisterWalkThroughWithContext(storepb.Engine_OCEANBASE, WalkThroughOmni)
}

// WalkThroughOmni performs DDL simulation using the omni MySQL catalog.
// Flow:
//  1. Create the catalog and select the target database.
//  2. loadWalkThroughCatalog: install each object individually with per-object
//     pseudo fallback, so one broken CREATE TABLE can't disable the whole
//     simulation.
//  3. catalog.Exec(userSQL) → execute user DDL
//  4. Map errors → *storepb.Advice
//  5. Convert updated catalog → DatabaseMetadata (for downstream rules)
func WalkThroughOmni(ctx schema.WalkThroughContext, d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
	if ctx.RawSQL == "" {
		return nil
	}

	dbName := d.GetProto().GetName()
	precheck := firstMySQLWalkThroughAdvice(
		precheckMySQLOmniWalkThrough(d, asts),
		precheckMySQLRawWalkThrough(d, ctx.RawSQL, asts),
	)

	// Step 1: Create the catalog and the target database.
	c := catalog.New()
	initSQL := fmt.Sprintf("SET foreign_key_checks = 0;\nCREATE DATABASE IF NOT EXISTS %s;\nUSE %s;", mysqlQuoteIdentifier(dbName), mysqlQuoteIdentifier(dbName))
	for _, targetDB := range mysqlRenameTargetDatabases(d, asts) {
		initSQL += fmt.Sprintf("\nCREATE DATABASE IF NOT EXISTS %s;", mysqlQuoteIdentifier(targetDB))
	}
	if _, err := c.Exec(initSQL, &catalog.ExecOptions{ContinueOnError: true}); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.DDLSimulationFailed.Int32(),
			Title:         "Failed to initialize catalog",
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Step 2: Install every schema object individually with pseudo fallback.
	// TODO: thread a real context.Context through WalkThroughContext; for now the
	// loader only uses it for early cancellation during catalog bulk-load.
	if err := loadWalkThroughCatalog(context.Background(), c, dbName, d.GetProto()); err != nil {
		return &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          code.DDLSimulationFailed.Int32(),
			Title:         "Failed to load schema",
			Content:       err.Error(),
			StartPosition: &storepb.Position{Line: 0},
		}
	}

	// Step 3: Execute user SQL.
	results, execErr := c.Exec(ctx.RawSQL, &catalog.ExecOptions{ContinueOnError: true})
	if execErr != nil {
		errCode := code.DDLSimulationFailed
		content := execErr.Error()
		var catErr *catalog.Error
		if errors.As(execErr, &catErr) {
			errCode = mapMySQLErrorToCode(catErr)
			content = mysqlCatalogErrorAdviceContent(d, catErr)
		}
		if strings.Contains(content, "BLOB, TEXT, GEOMETRY or JSON column ") {
			errCode = code.InvalidColumnDefault
			content = fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", mysqlCatalogErrorName(content))
		}
		execAdvice := mySQLWalkThroughAdvice{
			index: -1,
			advice: &storepb.Advice{
				Status:        storepb.Advice_ERROR,
				Code:          errCode.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			},
		}
		if precheck.beforeOrSame(execAdvice) {
			return precheck.advice
		}
		return execAdvice.advice
	}

	// Step 4: Report the first error from the simulation.
	var execAdvice mySQLWalkThroughAdvice
	for _, r := range results {
		if r.Error == nil {
			continue
		}
		errCode := mapMySQLErrorToCode(r.Error)
		content := r.Error.Error()
		if catErr, ok := r.Error.(*catalog.Error); ok {
			content = mysqlCatalogErrorAdviceContent(d, catErr)
		}
		if strings.Contains(content, "BLOB, TEXT, GEOMETRY or JSON column ") {
			errCode = code.InvalidColumnDefault
			content = fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", mysqlCatalogErrorName(content))
		}
		execAdvice = mySQLWalkThroughAdvice{
			index: r.Index,
			advice: &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    errCode.Int32(),
				Title:   content,
				Content: content,
				StartPosition: &storepb.Position{
					Line: int32(r.Line),
				},
			},
		}
		break
	}
	if precheck.beforeOrSame(execAdvice) {
		return precheck.advice
	}
	if execAdvice.advice != nil {
		return execAdvice.advice
	}

	// Step 5: Convert catalog state back to DatabaseMetadata for downstream rules.
	newProto := catalogToProto(c, dbName)
	newMetadata := model.NewDatabaseMetadata(newProto, nil, d.GetConfig(), storepb.Engine_MYSQL, true)
	d.ReplaceFrom(newMetadata)

	return nil
}

type mySQLWalkThroughAdvice struct {
	index  int
	advice *storepb.Advice
}

func firstMySQLWalkThroughAdvice(advices ...mySQLWalkThroughAdvice) mySQLWalkThroughAdvice {
	var result mySQLWalkThroughAdvice
	for _, advice := range advices {
		if advice.beforeOrSame(result) {
			result = advice
		}
	}
	return result
}

func (a mySQLWalkThroughAdvice) beforeOrSame(b mySQLWalkThroughAdvice) bool {
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
	aLine := mySQLWalkThroughAdviceLine(a.advice)
	bLine := mySQLWalkThroughAdviceLine(b.advice)
	if aLine <= 0 || bLine <= 0 {
		return true
	}
	return aLine <= bLine
}

func mySQLWalkThroughAdviceLine(advice *storepb.Advice) int32 {
	if advice == nil || advice.StartPosition == nil {
		return 0
	}
	return advice.StartPosition.Line
}

// Keep a token-aware fallback for CTAS forms that do not produce an omni AST yet.
func precheckMySQLRawWalkThrough(d *model.DatabaseMetadata, rawSQL string, asts []base.AST) mySQLWalkThroughAdvice {
	statements, err := mysqlparser.SplitSQL(rawSQL)
	if err != nil {
		return mySQLWalkThroughAdvice{}
	}
	statementIndex := 0
	for _, statement := range statements {
		if statement.Empty {
			continue
		}
		currentIndex := statementIndex
		statementIndex++
		text := strings.TrimSpace(statement.Text)
		if !isMySQLCreateTableAsStatement(text) {
			continue
		}
		if mySQLOmniASTSkipsCreateTableAsAdvice(d, asts, currentIndex) {
			continue
		}
		content := fmt.Sprintf("CREATE TABLE AS statement is used in \"%s\"", text)
		return mySQLWalkThroughAdvice{
			index: currentIndex,
			advice: &storepb.Advice{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementCreateTableAs.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: mySQLStatementStartPosition(statement.Start),
			},
		}
	}
	return mySQLWalkThroughAdvice{}
}

func mySQLOmniASTSkipsCreateTableAsAdvice(d *model.DatabaseMetadata, asts []base.AST, index int) bool {
	if index >= len(asts) {
		return false
	}
	node, ok := mysqlparser.GetOmniNode(asts[index])
	if !ok {
		return false
	}
	createTable, ok := node.(*ast.CreateTableStmt)
	return ok && createTable.Select != nil && createTable.IfNotExists && mySQLOmniTableExistsInCurrentDatabase(d, createTable.Table)
}

func isMySQLCreateTableAsStatement(statement string) bool {
	tokens := mySQLStatementTokens(statement)
	if len(tokens) < 3 || tokens[0] != "CREATE" {
		return false
	}
	idx := 1
	if tokens[idx] == "TEMPORARY" {
		idx++
	}
	if idx >= len(tokens) || tokens[idx] != "TABLE" {
		return false
	}
	return slices.Contains(tokens[idx+1:], "SELECT")
}

func mySQLStatementTokens(statement string) []string {
	var tokens []string
	for i := 0; i < len(statement); {
		switch statement[i] {
		case '\'', '"', '`':
			i = skipMySQLQuotedText(statement, i, statement[i])
		case '#':
			i = skipMySQLLineComment(statement, i+1)
		case '-':
			if i+1 < len(statement) && statement[i+1] == '-' {
				i = skipMySQLLineComment(statement, i+2)
			} else {
				i++
			}
		case '/':
			if i+1 < len(statement) && statement[i+1] == '*' {
				i = skipMySQLBlockComment(statement, i+2)
			} else {
				i++
			}
		default:
			if !isMySQLIdentifierChar(statement[i]) {
				i++
				continue
			}
			start := i
			for i < len(statement) && isMySQLIdentifierChar(statement[i]) {
				i++
			}
			tokens = append(tokens, strings.ToUpper(statement[start:i]))
		}
	}
	return tokens
}

func skipMySQLQuotedText(statement string, start int, quote byte) int {
	for i := start + 1; i < len(statement); i++ {
		if statement[i] == '\\' {
			i++
			continue
		}
		if statement[i] != quote {
			continue
		}
		if i+1 < len(statement) && statement[i+1] == quote {
			i++
			continue
		}
		return i + 1
	}
	return len(statement)
}

func skipMySQLLineComment(statement string, start int) int {
	for i := start; i < len(statement); i++ {
		if statement[i] == '\n' || statement[i] == '\r' {
			return i + 1
		}
	}
	return len(statement)
}

func skipMySQLBlockComment(statement string, start int) int {
	for i := start; i+1 < len(statement); i++ {
		if statement[i] == '*' && statement[i+1] == '/' {
			return i + 2
		}
	}
	return len(statement)
}

func isMySQLIdentifierChar(ch byte) bool {
	return ch == '_' || ch == '$' || ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func precheckMySQLOmniWalkThrough(d *model.DatabaseMetadata, asts []base.AST) mySQLWalkThroughAdvice {
	for i, unifiedAST := range asts {
		node, ok := mysqlparser.GetOmniNode(unifiedAST)
		if !ok {
			continue
		}
		text := ""
		if omniAST, ok := unifiedAST.(*mysqlparser.OmniAST); ok {
			text = strings.TrimSpace(omniAST.Text)
		}
		if advice := precheckMySQLOmniNode(d, node, text); advice != nil {
			advice.StartPosition = mySQLStatementStartPosition(unifiedAST.ASTStartPosition())
			return mySQLWalkThroughAdvice{index: i, advice: advice}
		}
	}
	return mySQLWalkThroughAdvice{}
}

func precheckMySQLOmniNode(d *model.DatabaseMetadata, node ast.Node, text string) *storepb.Advice {
	switch n := node.(type) {
	case *ast.CreateDatabaseStmt:
		if n.Name != "" && !isCurrentDatabase(d, n.Name) {
			return notCurrentMySQLDatabaseAdvice(d, n.Name)
		}
	case *ast.AlterDatabaseStmt:
		if n.Name != "" && !isCurrentDatabase(d, n.Name) {
			return notCurrentMySQLDatabaseAdvice(d, n.Name)
		}
	case *ast.DropDatabaseStmt:
		if n.Name != "" && !isCurrentDatabase(d, n.Name) {
			return notCurrentMySQLDatabaseAdvice(d, n.Name)
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
		if advice := precheckMySQLTableRefDatabase(d, n.Table); advice != nil {
			return advice
		}
		if n.IfNotExists && mySQLOmniTableExistsInCurrentDatabase(d, n.Table) {
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
		return precheckMySQLTableRefDatabase(d, n.Table)
	case *ast.DropTableStmt:
		for _, table := range n.Tables {
			if advice := precheckMySQLTableRefDatabase(d, table); advice != nil {
				return advice
			}
		}
	case *ast.CreateIndexStmt:
		return precheckMySQLTableRefDatabase(d, n.Table)
	case *ast.DropIndexStmt:
		return precheckMySQLTableRefDatabase(d, n.Table)
	case *ast.TruncateStmt:
		for _, table := range n.Tables {
			if advice := precheckMySQLTableRefDatabase(d, table); advice != nil {
				return advice
			}
		}
	case *ast.RenameTableStmt:
		for _, pair := range n.Pairs {
			if pair == nil {
				continue
			}
			if advice := precheckMySQLTableRefDatabase(d, pair.Old); advice != nil {
				return advice
			}
		}
	case *ast.CreateViewStmt:
		return precheckMySQLTableRefDatabase(d, n.Name)
	default:
	}
	return nil
}

func mysqlRenameTargetDatabases(d *model.DatabaseMetadata, asts []base.AST) []string {
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
		node, ok := mysqlparser.GetOmniNode(unifiedAST)
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

func mySQLStatementStartPosition(pos *storepb.Position) *storepb.Position {
	if pos == nil {
		return &storepb.Position{Line: 0}
	}
	return &storepb.Position{Line: pos.Line, Column: pos.Column}
}

func mysqlQuoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

func precheckMySQLTableRefDatabase(d *model.DatabaseMetadata, table *ast.TableRef) *storepb.Advice {
	if table == nil || table.Schema == "" || isCurrentDatabase(d, table.Schema) {
		return nil
	}
	return notCurrentMySQLDatabaseAdvice(d, table.Schema)
}

func mySQLOmniTableExistsInCurrentDatabase(d *model.DatabaseMetadata, table *ast.TableRef) bool {
	if table == nil || (table.Schema != "" && !isCurrentDatabase(d, table.Schema)) {
		return false
	}
	schema := d.GetSchemaMetadata("")
	return schema.GetTable(table.Name) != nil
}

func notCurrentMySQLDatabaseAdvice(d *model.DatabaseMetadata, databaseName string) *storepb.Advice {
	content := fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, d.DatabaseName())
	return &storepb.Advice{
		Status:        storepb.Advice_WARNING,
		Code:          code.NotCurrentDatabase.Int32(),
		Title:         content,
		Content:       content,
		StartPosition: &storepb.Position{Line: 0},
	}
}

// mapMySQLErrorToCode converts an omni MySQL catalog error to a bytebase error code.
func mapMySQLErrorToCode(err error) code.Code {
	catErr, ok := err.(*catalog.Error)
	if !ok {
		return code.DDLSimulationFailed
	}

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
	case catalog.ErrNoSuchColumn:
		return code.ColumnNotExists
	case catalog.ErrInvalidDefault:
		return code.InvalidColumnDefault
	case catalog.ErrDupKeyName, catalog.ErrDupIndex:
		return code.IndexExists
	case catalog.ErrMultiplePriKey:
		return code.PrimaryKeyExists
	case catalog.ErrDupEntry:
		return code.IndexExists
	case 1252:
		return code.SpatialIndexKeyNullable
	case catalog.ErrCantDropKey:
		return code.IndexNotExists
	case catalog.ErrIncorrectTableDefinition:
		return code.AutoIncrementExists
	case catalog.ErrInvalidOnUpdate:
		return code.OnUpdateColumnNotDatetimeOrTimestamp
	case catalog.ErrFKNoRefTable:
		return code.TableHasFK
	case catalog.ErrFKCannotDropParent:
		return code.TableHasFK
	case catalog.ErrFKMissingIndex, catalog.ErrFKIncompatibleColumns:
		return code.TableHasFK
	case catalog.ErrCheckConstraintViolated:
		return code.DDLSimulationFailed
	default:
		return code.DDLSimulationFailed
	}
}

func mysqlCatalogErrorAdviceContent(d *model.DatabaseMetadata, catErr *catalog.Error) string {
	name := mysqlCatalogErrorName(catErr.Message)
	switch catErr.Code {
	case catalog.ErrUnknownDatabase:
		return fmt.Sprintf("Database `%s` is not the current database `%s`", name, d.DatabaseName())
	case catalog.ErrDupTable:
		return fmt.Sprintf("Table `%s` already exists", mysqlUnqualifiedName(name))
	case catalog.ErrUnknownTable, catalog.ErrNoSuchTable:
		return fmt.Sprintf("Table `%s` does not exist", mysqlUnqualifiedName(name))
	case catalog.ErrDupColumn:
		return fmt.Sprintf("Column `%s` already exists", mysqlUnqualifiedName(name))
	case catalog.ErrNoSuchColumn:
		return fmt.Sprintf("Column `%s` does not exist", mysqlUnqualifiedName(name))
	case catalog.ErrInvalidDefault:
		return fmt.Sprintf("Invalid default value for column `%s`", mysqlUnqualifiedName(name))
	case catalog.ErrDupKeyName, catalog.ErrDupIndex, catalog.ErrDupEntry:
		return fmt.Sprintf("Index `%s` already exists", mysqlUnqualifiedName(name))
	case catalog.ErrCantDropKey:
		return fmt.Sprintf("Index `%s` does not exist", mysqlUnqualifiedName(name))
	default:
		return catErr.Message
	}
}

func mysqlCatalogErrorName(message string) string {
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

func mysqlUnqualifiedName(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// catalogToProto converts the omni catalog state to a storepb.DatabaseSchemaMetadata.
func catalogToProto(c *catalog.Catalog, dbName string) *storepb.DatabaseSchemaMetadata {
	dbMeta := &storepb.DatabaseSchemaMetadata{
		Name: dbName,
	}

	db := c.GetDatabase(dbName)
	if db == nil {
		return dbMeta
	}

	dbMeta.CharacterSet = db.Charset
	dbMeta.Collation = db.Collation

	// MySQL uses a single empty-name schema.
	schemaMeta := &storepb.SchemaMetadata{
		Name: "",
	}

	// Tables.
	tableNames := make([]string, 0, len(db.Tables))
	for name := range db.Tables {
		tableNames = append(tableNames, name)
	}
	slices.Sort(tableNames)

	for _, tName := range tableNames {
		t := db.Tables[tName]
		schemaMeta.Tables = append(schemaMeta.Tables, tableToProto(t))
	}

	// Views.
	viewNames := make([]string, 0, len(db.Views))
	for name := range db.Views {
		viewNames = append(viewNames, name)
	}
	slices.Sort(viewNames)

	for _, vName := range viewNames {
		v := db.Views[vName]
		schemaMeta.Views = append(schemaMeta.Views, &storepb.ViewMetadata{
			Name:       v.Name,
			Definition: v.Definition,
		})
	}

	// Functions.
	funcNames := make([]string, 0, len(db.Functions))
	for name := range db.Functions {
		funcNames = append(funcNames, name)
	}
	slices.Sort(funcNames)

	for _, fName := range funcNames {
		f := db.Functions[fName]
		schemaMeta.Functions = append(schemaMeta.Functions, routineToFunctionProto(f))
	}

	// Procedures.
	procNames := make([]string, 0, len(db.Procedures))
	for name := range db.Procedures {
		procNames = append(procNames, name)
	}
	slices.Sort(procNames)

	for _, pName := range procNames {
		p := db.Procedures[pName]
		schemaMeta.Procedures = append(schemaMeta.Procedures, routineToProcedureProto(p))
	}

	dbMeta.Schemas = append(dbMeta.Schemas, schemaMeta)
	return dbMeta
}

func tableToProto(t *catalog.Table) *storepb.TableMetadata {
	table := &storepb.TableMetadata{
		Name:      t.Name,
		Engine:    t.Engine,
		Charset:   t.Charset,
		Collation: t.Collation,
		Comment:   t.Comment,
	}

	// Columns.
	for _, col := range t.Columns {
		if col.Hidden == catalog.ColumnHiddenSystem {
			// System-hidden columns (e.g. backing columns of functional
			// indexes) are not reported by information_schema.
			continue
		}
		colMeta := &storepb.ColumnMetadata{
			Name:         col.Name,
			Position:     int32(col.Position),
			Type:         col.ColumnType,
			Nullable:     col.Nullable,
			CharacterSet: col.Charset,
			Collation:    col.Collation,
			Comment:      col.Comment,
		}
		switch {
		case col.AutoIncrement:
			// Match the driver convention: AUTO_INCREMENT columns report
			// "AUTO_INCREMENT" as the default (see backend/plugin/db/mysql/sync.go).
			colMeta.Default = "AUTO_INCREMENT"
		case col.Default != nil:
			colMeta.Default = normalizeOmniDefault(*col.Default, col.DefaultKind)
		case col.Nullable:
			// Match the driver convention: a nullable column without an explicit
			// default has an implicit default of NULL.
			colMeta.Default = "NULL"
		default:
		}
		if col.OnUpdate != "" {
			colMeta.OnUpdate = col.OnUpdate
		}
		if col.Generated != nil {
			genType := storepb.GenerationMetadata_TYPE_VIRTUAL
			if col.Generated.Stored {
				genType = storepb.GenerationMetadata_TYPE_STORED
			}
			colMeta.Generation = &storepb.GenerationMetadata{
				Type:       genType,
				Expression: col.Generated.Expr,
			}
		}
		table.Columns = append(table.Columns, colMeta)
	}

	// Indexes.
	for _, idx := range t.Indexes {
		indexType := strings.ToUpper(idx.IndexType)
		if indexType == "" {
			indexType = "BTREE"
		}
		idxMeta := &storepb.IndexMetadata{
			Name:    idx.Name,
			Type:    indexType,
			Unique:  idx.Unique,
			Primary: idx.Primary,
			Visible: idx.Visible,
			Comment: idx.Comment,
		}
		for _, col := range idx.Columns {
			if col.Expr != "" {
				idxMeta.Expressions = append(idxMeta.Expressions, col.Expr)
			} else {
				idxMeta.Expressions = append(idxMeta.Expressions, col.Name)
			}
			keyLength := int64(col.Length)
			if keyLength == 0 {
				// Match the driver convention: -1 means no prefix length specified.
				keyLength = -1
			}
			if keyLength == -1 && indexType == "SPATIAL" {
				// MySQL reports a key length of 32 for spatial index columns.
				keyLength = 32
			}
			idxMeta.KeyLength = append(idxMeta.KeyLength, keyLength)
			idxMeta.Descending = append(idxMeta.Descending, col.Descending)
		}
		table.Indexes = append(table.Indexes, idxMeta)
	}

	// Foreign keys from constraints.
	for _, con := range t.Constraints {
		switch con.Type {
		case catalog.ConForeignKey:
			fk := &storepb.ForeignKeyMetadata{
				Name:              con.Name,
				Columns:           con.Columns,
				ReferencedTable:   con.RefTable,
				ReferencedColumns: con.RefColumns,
				OnUpdate:          con.OnUpdate,
				OnDelete:          con.OnDelete,
				MatchType:         con.MatchType,
			}
			if con.RefDatabase != "" {
				fk.ReferencedSchema = con.RefDatabase
			}
			table.ForeignKeys = append(table.ForeignKeys, fk)
		case catalog.ConCheck:
			table.CheckConstraints = append(table.CheckConstraints, &storepb.CheckConstraintMetadata{
				Name:       con.Name,
				Expression: con.CheckExpr,
			})
		default:
		}
	}

	// Partition info.
	if t.Partitioning != nil {
		table.Partitions = partitionsToProto(t.Partitioning)
	}

	return table
}

// normalizeOmniDefault converts an omni catalog default value to the driver
// representation produced by SyncDBSchema (see backend/plugin/db/mysql/sync.go):
// static defaults are kept single-quoted for mysqldump compatibility and
// expression defaults are wrapped in parentheses.
func normalizeOmniDefault(value string, kind catalog.ColumnDefaultKind) string {
	switch kind {
	case catalog.ColumnDefaultConstant:
		if strings.EqualFold(value, "NULL") {
			return "NULL"
		}
		if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			return value
		}
		return fmt.Sprintf("'%s'", value)
	case catalog.ColumnDefaultExpression:
		return fmt.Sprintf("(%s)", value)
	default:
		return value
	}
}

func partitionsToProto(p *catalog.PartitionInfo) []*storepb.TablePartitionMetadata {
	expr := p.Expr
	if expr == "" && len(p.Columns) > 0 {
		// RANGE COLUMNS / LIST COLUMNS / KEY partitioning carries a column
		// list instead of an expression; the driver reports the joined list.
		expr = strings.Join(p.Columns, ",")
	}
	var result []*storepb.TablePartitionMetadata
	for _, pd := range p.Partitions {
		result = append(result, &storepb.TablePartitionMetadata{
			Name:       pd.Name,
			Type:       partitionTypeToProto(p.Type),
			Expression: expr,
			Value:      pd.ValueExpr,
		})
	}
	return result
}

func partitionTypeToProto(t string) storepb.TablePartitionMetadata_Type {
	switch strings.ToUpper(t) {
	case "RANGE":
		return storepb.TablePartitionMetadata_RANGE
	case "RANGE COLUMNS":
		return storepb.TablePartitionMetadata_RANGE_COLUMNS
	case "LIST":
		return storepb.TablePartitionMetadata_LIST
	case "LIST COLUMNS":
		return storepb.TablePartitionMetadata_LIST_COLUMNS
	case "HASH":
		return storepb.TablePartitionMetadata_HASH
	case "LINEAR HASH":
		return storepb.TablePartitionMetadata_LINEAR_HASH
	case "KEY":
		return storepb.TablePartitionMetadata_KEY
	case "LINEAR KEY":
		return storepb.TablePartitionMetadata_LINEAR_KEY
	default:
		return storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	}
}

func routineToFunctionProto(r *catalog.Routine) *storepb.FunctionMetadata {
	return &storepb.FunctionMetadata{
		Name:       r.Name,
		Definition: r.Body,
	}
}

func routineToProcedureProto(r *catalog.Routine) *storepb.ProcedureMetadata {
	return &storepb.ProcedureMetadata{
		Name:       r.Name,
		Definition: r.Body,
	}
}
