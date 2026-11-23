package omnicatalog

import (
	"context"
	"errors"
	"fmt"
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

// MapExecResults converts engine-specific omni catalog Exec results into []ExecResult.
func MapExecResults[R any](results []R, err error, project func(R) ExecResult) ([]ExecResult, error) {
	if err != nil {
		return nil, err
	}
	out := make([]ExecResult, len(results))
	for i, r := range results {
		out[i] = project(r)
	}
	return out, nil
}

// ExtractError builds an ExtractCatalogError callback for an engine catalog error type E.
func ExtractError[E error](extract func(E) (int, string)) func(error) (int, string, bool) {
	return func(err error) (int, string, bool) {
		var catErr E
		if errors.As(err, &catErr) {
			errCode, msg := extract(catErr)
			return errCode, msg, true
		}
		return 0, "", false
	}
}

// walkThroughAdvice pairs a statement index (-1 when unknown) with an advice.
type walkThroughAdvice struct {
	Index  int
	Advice *storepb.Advice
}

// firstWalkThroughAdvice returns the earliest non-nil advice by statement index and
// severity.
func firstWalkThroughAdvice(advices ...walkThroughAdvice) walkThroughAdvice {
	var result walkThroughAdvice
	for _, advice := range advices {
		if advice.beforeOrSame(result) {
			result = advice
		}
	}
	return result
}

// beforeOrSame reports whether a precedes or equals b in statement order (with
// higher-severity advice winning ties at the same statement index).
func (a walkThroughAdvice) beforeOrSame(b walkThroughAdvice) bool {
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
	if (a.Index >= 0) != (b.Index >= 0) {
		return a.Index >= 0
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

// InspectAST adapts a parser's GetNode and omni's ClassifyWalkThroughStatement
// into a WalkThroughEngine.InspectAST callback.
func InspectAST[Node any](
	getNode func(base.AST) (Node, bool),
	classify func(Node) omnianalysis.WalkThroughStatement,
) func(base.AST) (omnianalysis.WalkThroughStatement, bool) {
	return func(unifiedAST base.AST) (omnianalysis.WalkThroughStatement, bool) {
		node, ok := getNode(unifiedAST)
		if !ok {
			return omnianalysis.WalkThroughStatement{}, false
		}
		info := classify(node)
		if source, ok := unifiedAST.(interface{ ASTSource() string }); ok {
			info.Source = strings.TrimSpace(source.ASTSource())
		}
		return info, true
	}
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
	IsCTAS              func(statement string) bool
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

	precheck := precheckWalkThrough(d, asts, eng.InspectAST, eng.TrackBatchTables)
	if eng.SplitSQL != nil && eng.IsCTAS != nil {
		precheck = firstWalkThroughAdvice(
			precheck,
			precheckRawCTAS(d, rawSQL, asts, eng.SplitSQL, eng.IsCTAS, eng.InspectAST),
		)
	}

	c := eng.New()
	initSQL := "SET foreign_key_checks = 0;"
	for _, targetDB := range renameTargetDatabases(d, asts, eng.InspectAST) {
		initSQL += fmt.Sprintf("\nCREATE DATABASE IF NOT EXISTS %s;", quoteIdentifier(targetDB))
	}
	if _, err := eng.Exec(c, initSQL); err != nil {
		return DDLSimulationAdvice("Failed to initialize catalog", err)
	}

	if err := eng.Load(ctx, c, d.GetProto()); err != nil {
		return DDLSimulationAdvice("Failed to load schema", err)
	}

	results, execErr := eng.Exec(c, rawSQL)
	if execErr != nil {
		execAdvice := mapCatalogError(d, -1, &storepb.Position{Line: 0}, execErr, eng.ExtractCatalogError, eng.TiDBErrorDialect)
		if precheck.beforeOrSame(execAdvice) {
			return precheck.Advice
		}
		return execAdvice.Advice
	}

	var execAdvice walkThroughAdvice
	for _, r := range results {
		if r.Error == nil {
			continue
		}
		pos := &storepb.Position{Line: int32(r.Line)}
		if len(execASTs) > 0 {
			pos = execStatementPosition(execASTs, r.Index)
		}
		execAdvice = mapCatalogError(d, r.Index, pos, r.Error, eng.ExtractCatalogError, eng.TiDBErrorDialect)
		break
	}
	if precheck.beforeOrSame(execAdvice) {
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

func warningAdvice(c code.Code, content string, pos *storepb.Position) *storepb.Advice {
	return &storepb.Advice{
		Status:        storepb.Advice_WARNING,
		Code:          c.Int32(),
		Title:         content,
		Content:       content,
		StartPosition: pos,
	}
}

func execStatementPosition(execASTs []base.AST, index int) *storepb.Position {
	if index >= 0 && index < len(execASTs) {
		if pos := execASTs[index].ASTStartPosition(); pos != nil {
			return &storepb.Position{Line: pos.Line, Column: pos.Column}
		}
	}
	return &storepb.Position{Line: 0}
}

func statementStartPosition(pos *storepb.Position) *storepb.Position {
	if pos == nil {
		return &storepb.Position{Line: 0}
	}
	return &storepb.Position{Line: pos.Line, Column: pos.Column}
}

func quoteIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

func isCurrentDatabase(d *model.DatabaseMetadata, database string) bool {
	if !d.GetIsObjectCaseSensitive() {
		return strings.EqualFold(d.DatabaseName(), database)
	}
	return d.DatabaseName() == database
}

// renameTargetDatabases collects distinct non-current destination databases from
// RENAME TABLE / ALTER TABLE ... RENAME TO statements whose source is in the current
// database.
func renameTargetDatabases(d *model.DatabaseMetadata, asts []base.AST, inspectAST func(base.AST) (omnianalysis.WalkThroughStatement, bool)) []string {
	seen := make(map[string]bool)
	var targets []string
	for _, unifiedAST := range asts {
		info, ok := inspectAST(unifiedAST)
		if !ok {
			continue
		}
		for _, pair := range info.RenamePairs {
			if pair.Old.Schema != "" && !isCurrentDatabase(d, pair.Old.Schema) {
				continue
			}
			if pair.New.Schema == "" || isCurrentDatabase(d, pair.New.Schema) || seen[pair.New.Schema] {
				continue
			}
			seen[pair.New.Schema] = true
			targets = append(targets, pair.New.Schema)
		}
	}
	return targets
}

// precheckWalkThrough reports the first statement that the catalog would silently
// accept but Bytebase warns on: a cross-database reference, a dropped database, or a
// CREATE TABLE ... AS SELECT.
func precheckWalkThrough(d *model.DatabaseMetadata, asts []base.AST, inspectAST func(base.AST) (omnianalysis.WalkThroughStatement, bool), trackBatchTables bool) walkThroughAdvice {
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
			advice.StartPosition = statementStartPosition(unifiedAST.ASTStartPosition())
			return walkThroughAdvice{Index: i, Advice: advice}
		}
		if trackBatchTables {
			updateBatchTables(d, info, created)
		}
	}
	return walkThroughAdvice{}
}

func updateBatchTables(d *model.DatabaseMetadata, info omnianalysis.WalkThroughStatement, created map[string]bool) {
	switch info.Kind {
	case omnianalysis.WalkThroughStatementCreateTable:
		if len(info.TableRefs) > 0 {
			t := info.TableRefs[0]
			if t.Name != "" && (t.Schema == "" || isCurrentDatabase(d, t.Schema)) {
				created[tableLookupKey(d, t.Name)] = true
			}
		}
	case omnianalysis.WalkThroughStatementDropTable:
		for _, t := range info.TableRefs {
			if t.Name != "" && (t.Schema == "" || isCurrentDatabase(d, t.Schema)) {
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
	zeroPos := &storepb.Position{Line: 0}
	switch info.Kind {
	case omnianalysis.WalkThroughStatementCreateDatabase, omnianalysis.WalkThroughStatementAlterDatabase:
		if info.Database != "" && !isCurrentDatabase(d, info.Database) {
			return notCurrentDatabaseAdvice(d, info.Database)
		}
	case omnianalysis.WalkThroughStatementDropDatabase:
		if info.Database != "" && !isCurrentDatabase(d, info.Database) {
			return notCurrentDatabaseAdvice(d, info.Database)
		}
		if info.Database != "" {
			return warningAdvice(code.DatabaseIsDeleted, fmt.Sprintf("Database `%s` is deleted", info.Database), zeroPos)
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
			return warningAdvice(code.StatementCreateTableAs, fmt.Sprintf("CREATE TABLE AS statement is used in \"%s\"", info.Source), zeroPos)
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
	if table.Schema == "" || isCurrentDatabase(d, table.Schema) {
		return nil
	}
	return notCurrentDatabaseAdvice(d, table.Schema)
}

func tableExistsInBatch(d *model.DatabaseMetadata, created map[string]bool, table omnianalysis.TableRef) bool {
	if tableExistsInCurrentDatabase(d, table) {
		return true
	}
	if created == nil || table.Name == "" || (table.Schema != "" && !isCurrentDatabase(d, table.Schema)) {
		return false
	}
	return created[tableLookupKey(d, table.Name)]
}

func tableExistsInCurrentDatabase(d *model.DatabaseMetadata, table omnianalysis.TableRef) bool {
	if table.Name == "" || (table.Schema != "" && !isCurrentDatabase(d, table.Schema)) {
		return false
	}
	s := d.GetSchemaMetadata("")
	return s != nil && s.GetTable(table.Name) != nil
}

func notCurrentDatabaseAdvice(d *model.DatabaseMetadata, databaseName string) *storepb.Advice {
	return warningAdvice(
		code.NotCurrentDatabase,
		fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, d.DatabaseName()),
		&storepb.Position{Line: 0},
	)
}

// precheckRawCTAS provides a token-aware fallback for CTAS forms that do not produce
// an omni AST yet.
func precheckRawCTAS(
	d *model.DatabaseMetadata,
	rawSQL string,
	asts []base.AST,
	splitSQL func(string) ([]base.Statement, error),
	isCTAS func(string) bool,
	inspectAST func(base.AST) (omnianalysis.WalkThroughStatement, bool),
) walkThroughAdvice {
	statements, err := splitSQL(rawSQL)
	if err != nil {
		return walkThroughAdvice{}
	}
	statementIndex := 0
	for _, statement := range statements {
		if statement.Empty {
			continue
		}
		currentIndex := statementIndex
		statementIndex++
		text := strings.TrimSpace(statement.Text)
		if !isCTAS(text) || astSkipsCreateTableAsAdvice(d, asts, currentIndex, inspectAST) {
			continue
		}
		return walkThroughAdvice{
			Index: currentIndex,
			Advice: warningAdvice(
				code.StatementCreateTableAs,
				fmt.Sprintf("CREATE TABLE AS statement is used in \"%s\"", text),
				statementStartPosition(statement.Start),
			),
		}
	}
	return walkThroughAdvice{}
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

// MySQL-family error numbers shared across omni/{mysql,mariadb,oceanbase,tidb}/catalog.
const (
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
)

func mapCatalogError(
	d *model.DatabaseMetadata,
	index int,
	position *storepb.Position,
	err error,
	extract func(error) (int, string, bool),
	tidbDialect bool,
) walkThroughAdvice {
	errCode := code.DDLSimulationFailed
	content := err.Error()
	if catCode, catMsg, ok := extract(err); ok {
		errCode, content = mapMySQLFamilyCatalogError(d, catCode, catMsg, tidbDialect)
	}
	if !tidbDialect && strings.Contains(content, "BLOB, TEXT, GEOMETRY or JSON column ") {
		errCode = code.InvalidColumnDefault
		content = fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", catalogErrorName(content))
	}
	return walkThroughAdvice{
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
	case errInvalidDefault:
		return code.InvalidColumnDefault, fmt.Sprintf("Invalid default value for column `%s`", unqualifiedName(name))
	case errBlobCantHaveDefault:
		if tidbDialect {
			return code.InvalidColumnDefault, fmt.Sprintf("BLOB, TEXT or JSON column `%s` can't have a default value", unqualifiedName(name))
		}
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
	default:
	}
	return code.DDLSimulationFailed, catMsg
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
