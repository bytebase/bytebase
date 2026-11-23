package omnicatalog

import (
	"context"
	"fmt"
	"slices"
	"strings"

	omnianalysis "github.com/bytebase/omni/analysis"
	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// ExecResult is the engine-neutral projection of an omni catalog execution result.
type ExecResult struct {
	Index int
	Line  int
	Error error
}

// WalkThroughAdvice pairs a statement index (-1 when unknown) with an advice.
type WalkThroughAdvice struct {
	Index  int
	Advice *storepb.Advice
}

// FirstWalkThroughAdvice returns the earliest non-nil advice by statement index and
// severity.
func FirstWalkThroughAdvice(advices ...WalkThroughAdvice) WalkThroughAdvice {
	var result WalkThroughAdvice
	for _, advice := range advices {
		if advice.BeforeOrSame(result) {
			result = advice
		}
	}
	return result
}

// BeforeOrSame reports whether a precedes or equals b in statement order (with
// higher-severity advice winning ties at the same statement index).
func (a WalkThroughAdvice) BeforeOrSame(b WalkThroughAdvice) bool {
	if a.Advice == nil {
		return false
	}
	if b.Advice == nil {
		return true
	}
	if a.Index >= 0 && b.Index >= 0 {
		if a.Index != b.Index {
			return a.Index < b.Index
		}
		if a.Advice.Status != b.Advice.Status {
			return a.Advice.Status > b.Advice.Status
		}
		return true
	}
	if a.Index >= 0 && b.Index < 0 {
		return true
	}
	if a.Index < 0 && b.Index >= 0 {
		return false
	}
	aLine := walkThroughAdviceLine(a.Advice)
	bLine := walkThroughAdviceLine(b.Advice)
	if aLine <= 0 || bLine <= 0 {
		return true
	}
	return aLine <= bLine
}

func walkThroughAdviceLine(advice *storepb.Advice) int32 {
	if advice == nil || advice.StartPosition == nil {
		return 0
	}
	return advice.StartPosition.Line
}

// WalkThroughEngine adapts an omni MySQL-family catalog for DDL walk-through.
type WalkThroughEngine[Catalog any] struct {
	Engine              storepb.Engine
	New                 func() *Catalog
	Load                func(ctx context.Context, c *Catalog, meta *metadatapb.DatabaseSchemaMetadata) error
	Exec                func(c *Catalog, sql string) ([]ExecResult, error)
	DatabaseMetadata    func(c *Catalog, dbName string) *metadatapb.DatabaseSchemaMetadata
	InspectAST          func(a base.AST) (omnianalysis.WalkThroughStatement, bool)
	ExtractCatalogError func(err error) (errCode int, message string, ok bool)
	SplitSQL            func(rawSQL string) ([]base.Statement, error)
	TrackBatchTables    bool
	TiDBErrorDialect    bool
	CaseSensitiveResult func(d *model.DatabaseMetadata) bool
}

// RunWalkThrough simulates rawSQL against d using the provided MySQL-family omni catalog.
// When execASTs is non-empty, per-statement execution errors are positioned from
// execASTs[r.Index] instead of r.Line (used when rawSQL is reconstructed from ASTs).
func RunWalkThrough[Catalog any](
	ctx context.Context,
	eng WalkThroughEngine[Catalog],
	rawSQL string,
	d *model.DatabaseMetadata,
	asts []base.AST,
	execASTs []base.AST,
) *storepb.Advice {
	if strings.TrimSpace(rawSQL) == "" {
		return nil
	}
	dbName := d.GetProto().GetName()

	precheck := PrecheckWalkThrough(d, asts, eng.InspectAST, eng.TrackBatchTables)
	if eng.SplitSQL != nil {
		precheck = FirstWalkThroughAdvice(
			precheck,
			PrecheckRawCTAS(d, rawSQL, asts, eng.SplitSQL, eng.InspectAST),
		)
	}

	c := eng.New()
	initSQL := "SET foreign_key_checks = 0;"
	for _, targetDB := range RenameTargetDatabases(d, asts, eng.InspectAST) {
		initSQL += fmt.Sprintf("\nCREATE DATABASE IF NOT EXISTS %s;", QuoteIdentifier(targetDB))
	}
	if _, err := eng.Exec(c, initSQL); err != nil {
		return DDLSimulationAdvice("Failed to initialize catalog", err)
	}

	if err := eng.Load(ctx, c, d.GetProto()); err != nil {
		return DDLSimulationAdvice("Failed to load schema", err)
	}

	results, execErr := eng.Exec(c, rawSQL)
	if execErr != nil {
		execAdvice := MapCatalogError(d, -1, &storepb.Position{Line: 0}, execErr, eng.ExtractCatalogError, eng.TiDBErrorDialect)
		if precheck.BeforeOrSame(execAdvice) {
			return precheck.Advice
		}
		return execAdvice.Advice
	}

	var execAdvice WalkThroughAdvice
	for _, r := range results {
		if r.Error == nil {
			continue
		}
		pos := &storepb.Position{Line: int32(r.Line)}
		if len(execASTs) > 0 {
			pos = ExecStatementPosition(execASTs, r.Index)
		}
		execAdvice = MapCatalogError(d, r.Index, pos, r.Error, eng.ExtractCatalogError, eng.TiDBErrorDialect)
		break
	}
	if precheck.BeforeOrSame(execAdvice) {
		return precheck.Advice
	}
	if execAdvice.Advice != nil {
		return execAdvice.Advice
	}

	caseSensitive := true
	if eng.CaseSensitiveResult != nil {
		caseSensitive = eng.CaseSensitiveResult(d)
	}
	newProto := eng.DatabaseMetadata(c, dbName)
	newMetadata := model.NewDatabaseMetadata(newProto, nil, d.GetConfig(), eng.Engine, caseSensitive)
	d.ReplaceFrom(newMetadata)
	return nil
}

// DDLSimulationAdvice constructs an ERROR advice for catalog setup/load failures.
func DDLSimulationAdvice(title string, err error) *storepb.Advice {
	return &storepb.Advice{
		Status:        storepb.Advice_ERROR,
		Code:          code.DDLSimulationFailed.Int32(),
		Title:         title,
		Content:       err.Error(),
		StartPosition: &storepb.Position{Line: 0},
	}
}

// ExecStatementPosition maps a catalog result index back to the originating AST's
// start position.
func ExecStatementPosition(execASTs []base.AST, index int) *storepb.Position {
	if index >= 0 && index < len(execASTs) {
		if pos := execASTs[index].ASTStartPosition(); pos != nil {
			return &storepb.Position{Line: pos.Line, Column: pos.Column}
		}
	}
	return &storepb.Position{Line: 0}
}

// StatementStartPosition returns a copy of pos or line 0 when nil.
func StatementStartPosition(pos *storepb.Position) *storepb.Position {
	if pos == nil {
		return &storepb.Position{Line: 0}
	}
	return &storepb.Position{Line: pos.Line, Column: pos.Column}
}

// QuoteIdentifier quotes a MySQL-family identifier with backticks.
func QuoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

// IsCurrentDatabase reports whether database matches d's current database name,
// honoring d's object case sensitivity.
func IsCurrentDatabase(d *model.DatabaseMetadata, database string) bool {
	if !d.GetIsObjectCaseSensitive() {
		return strings.EqualFold(d.DatabaseName(), database)
	}
	return d.DatabaseName() == database
}

// RenameTargetDatabases collects distinct non-current destination databases from
// RENAME TABLE / ALTER TABLE ... RENAME TO statements whose source is in the current
// database.
func RenameTargetDatabases(d *model.DatabaseMetadata, asts []base.AST, inspectAST func(base.AST) (omnianalysis.WalkThroughStatement, bool)) []string {
	seen := make(map[string]bool)
	var targets []string
	for _, unifiedAST := range asts {
		info, ok := inspectAST(unifiedAST)
		if !ok {
			continue
		}
		for _, pair := range info.RenamePairs {
			if pair.Old.Schema != "" && !IsCurrentDatabase(d, pair.Old.Schema) {
				continue
			}
			if pair.New.Schema == "" || IsCurrentDatabase(d, pair.New.Schema) || seen[pair.New.Schema] {
				continue
			}
			seen[pair.New.Schema] = true
			targets = append(targets, pair.New.Schema)
		}
	}
	return targets
}

// PrecheckWalkThrough reports the first statement that the catalog would silently
// accept but Bytebase warns on: a cross-database reference, a dropped database, or a
// CREATE TABLE ... AS SELECT.
func PrecheckWalkThrough(d *model.DatabaseMetadata, asts []base.AST, inspectAST func(base.AST) (omnianalysis.WalkThroughStatement, bool), trackBatchTables bool) WalkThroughAdvice {
	var created map[string]bool
	if trackBatchTables {
		created = make(map[string]bool)
	}
	for i, unifiedAST := range asts {
		info, ok := inspectAST(unifiedAST)
		if !ok {
			continue
		}
		if advice := precheckStatement(d, info, created); advice != nil {
			advice.StartPosition = StatementStartPosition(unifiedAST.ASTStartPosition())
			return WalkThroughAdvice{Index: i, Advice: advice}
		}
		if trackBatchTables {
			updateBatchTables(d, info, created)
		}
	}
	return WalkThroughAdvice{}
}

func updateBatchTables(d *model.DatabaseMetadata, info omnianalysis.WalkThroughStatement, created map[string]bool) {
	switch info.Kind {
	case omnianalysis.WalkThroughStatementCreateTable:
		if len(info.TableRefs) > 0 {
			t := info.TableRefs[0]
			if t.Name != "" && (t.Schema == "" || IsCurrentDatabase(d, t.Schema)) {
				created[tableLookupKey(d, t.Name)] = true
			}
		}
	case omnianalysis.WalkThroughStatementDropTable:
		for _, t := range info.TableRefs {
			if t.Name != "" && (t.Schema == "" || IsCurrentDatabase(d, t.Schema)) {
				delete(created, tableLookupKey(d, t.Name))
			}
		}
	default:
	}
}

func tableLookupKey(d *model.DatabaseMetadata, name string) string {
	if !d.GetIsObjectCaseSensitive() {
		return strings.ToLower(name)
	}
	return name
}

func precheckStatement(d *model.DatabaseMetadata, info omnianalysis.WalkThroughStatement, created map[string]bool) *storepb.Advice {
	switch info.Kind {
	case omnianalysis.WalkThroughStatementCreateDatabase, omnianalysis.WalkThroughStatementAlterDatabase:
		if info.Database != "" && !IsCurrentDatabase(d, info.Database) {
			return notCurrentDatabaseAdvice(d, info.Database)
		}
	case omnianalysis.WalkThroughStatementDropDatabase:
		if info.Database != "" && !IsCurrentDatabase(d, info.Database) {
			return notCurrentDatabaseAdvice(d, info.Database)
		}
		if info.Database != "" {
			content := fmt.Sprintf("Database `%s` is deleted", info.Database)
			return &storepb.Advice{
				Status:        storepb.Advice_WARNING,
				Code:          code.DatabaseIsDeleted.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
		}
	case omnianalysis.WalkThroughStatementCreateTable:
		if len(info.TableRefs) > 0 {
			if advice := precheckTableRefDatabase(d, info.TableRefs[0]); advice != nil {
				return advice
			}
			if info.IfNotExists && tableExistsInBatch(d, created, info.TableRefs[0]) {
				return nil
			}
		}
		if info.IsCTAS {
			content := fmt.Sprintf("CREATE TABLE AS statement is used in \"%s\"", info.Source)
			return &storepb.Advice{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementCreateTableAs.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: &storepb.Position{Line: 0},
			}
		}
	case omnianalysis.WalkThroughStatementDropTable, omnianalysis.WalkThroughStatementTableRefCheck, omnianalysis.WalkThroughStatementRenameTable:
		for _, ref := range info.TableRefs {
			if advice := precheckTableRefDatabase(d, ref); advice != nil {
				return advice
			}
		}
	default:
	}
	return nil
}

func precheckTableRefDatabase(d *model.DatabaseMetadata, table omnianalysis.TableRef) *storepb.Advice {
	if table.Schema == "" || IsCurrentDatabase(d, table.Schema) {
		return nil
	}
	return notCurrentDatabaseAdvice(d, table.Schema)
}

func tableExistsInBatch(d *model.DatabaseMetadata, created map[string]bool, table omnianalysis.TableRef) bool {
	if tableExistsInCurrentDatabase(d, table) {
		return true
	}
	if created == nil || table.Name == "" || (table.Schema != "" && !IsCurrentDatabase(d, table.Schema)) {
		return false
	}
	return created[tableLookupKey(d, table.Name)]
}

func tableExistsInCurrentDatabase(d *model.DatabaseMetadata, table omnianalysis.TableRef) bool {
	if table.Name == "" || (table.Schema != "" && !IsCurrentDatabase(d, table.Schema)) {
		return false
	}
	s := d.GetSchemaMetadata("")
	if s == nil {
		return false
	}
	return s.GetTable(table.Name) != nil
}

func notCurrentDatabaseAdvice(d *model.DatabaseMetadata, databaseName string) *storepb.Advice {
	content := fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, d.DatabaseName())
	return &storepb.Advice{
		Status:        storepb.Advice_WARNING,
		Code:          code.NotCurrentDatabase.Int32(),
		Title:         content,
		Content:       content,
		StartPosition: &storepb.Position{Line: 0},
	}
}

// PrecheckRawCTAS provides a token-aware fallback for CTAS forms that do not produce
// an omni AST yet.
func PrecheckRawCTAS(
	d *model.DatabaseMetadata,
	rawSQL string,
	asts []base.AST,
	splitSQL func(string) ([]base.Statement, error),
	inspectAST func(base.AST) (omnianalysis.WalkThroughStatement, bool),
) WalkThroughAdvice {
	statements, err := splitSQL(rawSQL)
	if err != nil {
		return WalkThroughAdvice{}
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
		if astSkipsCreateTableAsAdvice(d, asts, currentIndex, inspectAST) {
			continue
		}
		content := fmt.Sprintf("CREATE TABLE AS statement is used in \"%s\"", text)
		return WalkThroughAdvice{
			Index: currentIndex,
			Advice: &storepb.Advice{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementCreateTableAs.Int32(),
				Title:         content,
				Content:       content,
				StartPosition: StatementStartPosition(statement.Start),
			},
		}
	}
	return WalkThroughAdvice{}
}

func astSkipsCreateTableAsAdvice(d *model.DatabaseMetadata, asts []base.AST, index int, inspectAST func(base.AST) (omnianalysis.WalkThroughStatement, bool)) bool {
	if index >= len(asts) {
		return false
	}
	info, ok := inspectAST(asts[index])
	if !ok || info.Kind != omnianalysis.WalkThroughStatementCreateTable || !info.IsCTAS || !info.IfNotExists || len(info.TableRefs) == 0 {
		return false
	}
	return tableExistsInCurrentDatabase(d, info.TableRefs[0])
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

// MySQL-family error numbers shared across omni/{mysql,oceanbase,tidb}/catalog.
const (
	errDupDatabase              = 1007
	errUnknownDatabase          = 1049
	errDupTable                 = 1050
	errUnknownTable             = 1051
	errNoSuchColumn             = 1054
	errDupColumn                = 1060
	errDupKeyName               = 1061
	errDupEntry                 = 1062
	errInvalidDefault           = 1067
	errMultiplePriKey           = 1068
	errKeyColumnDoesNotExist    = 1072
	errIncorrectTableDefinition = 1075
	errCantDropKey              = 1091
	errBlobCantHaveDefault      = 1101
	errNoSuchTable              = 1146
	errSpatialIndexKeyNullable  = 1252
	errInvalidOnUpdate          = 1294
	errFKMissingIndex           = 1822
	errFKNoRefTable             = 1824
	errDupIndex                 = 1831
	errFKCannotDropParent       = 3730
	errFKIncompatibleColumns    = 3780
	errCheckConstraintViolated  = 3819
)

// MapCatalogError converts an omni MySQL-family catalog execution error into a
// WalkThroughAdvice.
func MapCatalogError(
	d *model.DatabaseMetadata,
	index int,
	position *storepb.Position,
	err error,
	extract func(error) (int, string, bool),
	tidbDialect bool,
) WalkThroughAdvice {
	errCode := code.DDLSimulationFailed
	content := err.Error()
	if catCode, catMsg, ok := extract(err); ok {
		errCode, content = mapMySQLFamilyCatalogError(d, catCode, catMsg, tidbDialect)
	}
	if !tidbDialect && strings.Contains(content, "BLOB, TEXT, GEOMETRY or JSON column ") {
		errCode = code.InvalidColumnDefault
		content = fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", catalogErrorName(content))
	}
	return WalkThroughAdvice{
		Index: index,
		Advice: &storepb.Advice{
			Status:        storepb.Advice_ERROR,
			Code:          errCode.Int32(),
			Title:         content,
			Content:       content,
			StartPosition: position,
		},
	}
}

func mapMySQLFamilyCatalogError(d *model.DatabaseMetadata, catCode int, catMsg string, tidbDialect bool) (code.Code, string) {
	name := catalogErrorName(catMsg)
	switch catCode {
	case errDupDatabase:
		return code.DDLSimulationFailed, catMsg
	case errUnknownDatabase:
		return code.NotCurrentDatabase, fmt.Sprintf("Database `%s` is not the current database `%s`", name, d.DatabaseName())
	case errDupTable:
		return code.TableExists, fmt.Sprintf("Table `%s` already exists", unqualifiedName(name))
	case errUnknownTable, errNoSuchTable:
		return code.TableNotExists, fmt.Sprintf("Table `%s` does not exist", unqualifiedName(name))
	case errDupColumn:
		return code.ColumnExists, fmt.Sprintf("Column `%s` already exists", unqualifiedName(name))
	case errNoSuchColumn:
		return code.ColumnNotExists, fmt.Sprintf("Column `%s` does not exist", unqualifiedName(name))
	case errKeyColumnDoesNotExist:
		if tidbDialect {
			return code.ColumnNotExists, fmt.Sprintf("Column `%s` does not exist", strings.TrimPrefix(catMsg, "column does not exist: "))
		}
		return code.DDLSimulationFailed, catMsg
	case errInvalidDefault:
		return code.InvalidColumnDefault, fmt.Sprintf("Invalid default value for column `%s`", unqualifiedName(name))
	case errBlobCantHaveDefault:
		if tidbDialect {
			return code.InvalidColumnDefault, fmt.Sprintf("BLOB, TEXT or JSON column `%s` can't have a default value", unqualifiedName(name))
		}
		return code.DDLSimulationFailed, catMsg
	case errDupKeyName, errDupIndex, errDupEntry:
		return code.IndexExists, fmt.Sprintf("Index `%s` already exists", unqualifiedName(name))
	case errMultiplePriKey:
		if tidbDialect {
			return code.PrimaryKeyExists, "Multiple primary key defined"
		}
		return code.PrimaryKeyExists, catMsg
	case errSpatialIndexKeyNullable:
		if !tidbDialect {
			return code.SpatialIndexKeyNullable, catMsg
		}
		return code.DDLSimulationFailed, catMsg
	case errCantDropKey:
		if tidbDialect && strings.HasPrefix(catMsg, "Can't DROP") {
			return code.ColumnNotExists, fmt.Sprintf("Column `%s` does not exist", tidbCantDropName(catMsg))
		}
		if tidbDialect {
			return code.IndexNotExists, fmt.Sprintf("Index `%s` does not exist", tidbDroppedIndexName(catMsg))
		}
		return code.IndexNotExists, fmt.Sprintf("Index `%s` does not exist", unqualifiedName(name))
	case errIncorrectTableDefinition:
		if tidbDialect {
			return code.AutoIncrementExists, "There can be only one auto column and it must be defined as a key"
		}
		return code.AutoIncrementExists, catMsg
	case errInvalidOnUpdate:
		if tidbDialect {
			return code.OnUpdateColumnNotDatetimeOrTimestamp, fmt.Sprintf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", unqualifiedName(name))
		}
		return code.OnUpdateColumnNotDatetimeOrTimestamp, catMsg
	case errFKNoRefTable, errFKCannotDropParent, errFKMissingIndex, errFKIncompatibleColumns:
		if !tidbDialect {
			return code.TableHasFK, catMsg
		}
		return code.DDLSimulationFailed, catMsg
	case errCheckConstraintViolated:
		return code.DDLSimulationFailed, catMsg
	default:
		return code.DDLSimulationFailed, catMsg
	}
}

func catalogErrorName(message string) string {
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

func unqualifiedName(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// tidbCantDropName extracts <col> from "Can't DROP '<col>'; check that ...".
func tidbCantDropName(message string) string {
	rest, ok := strings.CutPrefix(message, "Can't DROP '")
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
