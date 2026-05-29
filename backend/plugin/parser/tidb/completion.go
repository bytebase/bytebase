package tidb

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bytebase/omni/tidb/catalog"
	omnicompletion "github.com/bytebase/omni/tidb/completion"
	tidbparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// genericColumnType is a syntactically valid fallback type used when a column's
// real type is empty or fails to parse, so the column name still surfaces as a
// completion candidate.
const genericColumnType = "int"

// writeStatementKeywords are statement-initiating keywords for write, DDL,
// transaction-control, and admin statements. They are dropped from completion
// in the read-only query scene, leaving only read statements (SELECT, WITH,
// EXPLAIN, DESC/DESCRIBE, SHOW, HELP, TABLE, VALUES) and clause keywords.
var writeStatementKeywords = map[string]bool{
	// DML.
	"INSERT": true, "UPDATE": true, "DELETE": true, "REPLACE": true,
	"LOAD": true, "BATCH": true,
	// DDL.
	"CREATE": true, "ALTER": true, "DROP": true, "TRUNCATE": true, "RENAME": true,
	// Privileges / session.
	"GRANT": true, "REVOKE": true, "SET": true, "USE": true,
	// Transaction control.
	"BEGIN": true, "START": true, "COMMIT": true, "ROLLBACK": true,
	"SAVEPOINT": true, "RELEASE": true, "XA": true,
	// Admin / maintenance.
	"ANALYZE": true, "OPTIMIZE": true, "FLUSH": true, "REPAIR": true,
	"CHECK": true, "RESET": true, "KILL": true, "LOCK": true, "UNLOCK": true,
	// Prepared statements / routines.
	"PREPARE": true, "EXECUTE": true, "DEALLOCATE": true, "CALL": true, "DO": true,
}

func init() {
	base.RegisterCompleteFunc(storepb.Engine_TIDB, Completion)
}

// Completion provides auto-complete candidates for TiDB statements using the
// omni TiDB completion engine, replacing the previous mysql ANTLR completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	cat := buildCatalog(ctx, cCtx, statement)
	pos := lineOffsetToBytePos(statement, caretLine, caretOffset)

	caretInBacktick := caretInsideBacktickIdentifier(statement, pos)
	candidateMap := make(map[string]base.Candidate)
	for _, c := range omnicompletion.Complete(statement, pos, cat) {
		t := omniCandidateTypeToBase(c.Type)
		text := c.Text
		if isObjectIdentifierCandidate(t) {
			text = quoteIdentifierIfNeeded(c.Text, caretInBacktick)
		}
		candidateMap[string(t)+":"+text] = base.Candidate{
			Type:       t,
			Text:       text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}
	}

	// In the read-only query scene, drop keywords that initiate writes
	// (DML/DDL/transaction/admin statements) so the editor only suggests
	// read statements.
	if cCtx.Scene == base.SceneTypeQuery {
		for key, c := range candidateMap {
			if c.Type == base.CandidateTypeKeyword && writeStatementKeywords[strings.ToUpper(c.Text)] {
				delete(candidateMap, key)
			}
		}
	}

	result := make([]base.Candidate, 0, len(candidateMap))
	for _, c := range candidateMap {
		result = append(result, c)
	}
	slices.SortFunc(result, func(a, b base.Candidate) int {
		if a.Type != b.Type {
			if a.Type < b.Type {
				return -1
			}
			return 1
		}
		if a.Text < b.Text {
			return -1
		}
		if a.Text > b.Text {
			return 1
		}
		return 0
	})
	return result, nil
}

// buildCatalog constructs an omni TiDB catalog from Bytebase metadata by
// replaying minimal DDL. It fully loads the default database plus any database
// referenced as a qualifier in the statement (so cross-database qualified
// completion like `other_db.tbl.col` resolves), and registers every other known
// database name so database-name candidates still surface. Every identifier is
// backticked so reserved words and special characters parse; tables are created
// one at a time so a single unparseable column type cannot empty the catalog.
func buildCatalog(ctx context.Context, cCtx base.CompletionContext, statement string) *catalog.Catalog {
	cat := catalog.New()
	if cCtx.Metadata == nil || cCtx.DefaultDatabase == "" {
		return cat
	}

	// Fully load the default database plus any qualifier-referenced database.
	loadOrder := []string{cCtx.DefaultDatabase}
	seen := map[string]bool{cCtx.DefaultDatabase: true}
	allNames := listAllDatabaseNames(ctx, cCtx)
	for _, name := range allNames {
		if !seen[name] && statementReferencesDatabase(statement, name) {
			loadOrder = append(loadOrder, name)
			seen[name] = true
		}
	}

	// Register the remaining known database names (name only) so they surface as
	// database candidates even when not fully loaded.
	var reg strings.Builder
	for _, name := range allNames {
		if !seen[name] {
			reg.WriteString("CREATE DATABASE " + backtickIdentifier(name) + "; ")
		}
	}
	if reg.Len() > 0 {
		_, _ = cat.Exec(reg.String(), &catalog.ExecOptions{ContinueOnError: true})
	}

	for _, name := range loadOrder {
		loadDatabaseObjects(ctx, cCtx, cat, name)
	}

	// Restore the current database to the default so unqualified references
	// resolve against it.
	_, _ = cat.Exec("USE "+backtickIdentifier(cCtx.DefaultDatabase)+";", &catalog.ExecOptions{ContinueOnError: true})
	return cat
}

// loadDatabaseObjects fully loads one database's tables and views into the
// catalog, under that database's namespace.
func loadDatabaseObjects(ctx context.Context, cCtx base.CompletionContext, cat *catalog.Catalog, dbName string) {
	db := backtickIdentifier(dbName)
	_, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, dbName)
	if err != nil || dbMeta == nil {
		// Still register the name so it can be a database candidate.
		_, _ = cat.Exec("CREATE DATABASE "+db+";", &catalog.ExecOptions{ContinueOnError: true})
		return
	}
	if _, err := cat.Exec("CREATE DATABASE "+db+"; USE "+db+";", &catalog.ExecOptions{ContinueOnError: true}); err != nil {
		return
	}
	schema := dbMeta.GetSchemaMetadata("")
	if schema == nil {
		return
	}
	for _, tableName := range schema.ListTableNames() {
		if table := schema.GetTable(tableName); table != nil {
			defineTable(cat, tableName, table.GetProto().GetColumns())
		}
	}
	for _, viewName := range schema.ListViewNames() {
		if view := schema.GetView(viewName); view != nil {
			defineView(cat, viewName, view.GetDefinition())
		}
	}
}

// listAllDatabaseNames returns the instance's database names, or nil if the
// completion context cannot enumerate them.
func listAllDatabaseNames(ctx context.Context, cCtx base.CompletionContext) []string {
	if cCtx.ListDatabaseNames == nil {
		return nil
	}
	names, err := cCtx.ListDatabaseNames(ctx, cCtx.InstanceID)
	if err != nil {
		return nil
	}
	return names
}

// statementReferencesDatabase reports whether dbName appears as a database
// qualifier (`dbName.`) in the statement, at an identifier boundary.
func statementReferencesDatabase(statement, dbName string) bool {
	if dbName == "" {
		return false
	}
	s := strings.ToLower(statement)
	name := strings.ToLower(dbName)
	for _, q := range []string{name + ".", "`" + name + "`."} {
		from := 0
		for {
			i := strings.Index(s[from:], q)
			if i < 0 {
				break
			}
			at := from + i
			if at == 0 || !isIdentByte(s[at-1]) {
				return true
			}
			from = at + 1
		}
	}
	return false
}

func isIdentByte(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

// defineTable installs a table in the catalog, retrying with generic column
// types if the real types fail to parse so that column names still surface.
func defineTable(cat *catalog.Catalog, name string, columns []*storepb.ColumnMetadata) {
	if len(columns) == 0 {
		return
	}
	if execTableDDL(cat, name, columns, false) {
		return
	}
	execTableDDL(cat, name, columns, true)
}

func execTableDDL(cat *catalog.Catalog, name string, columns []*storepb.ColumnMetadata, generic bool) bool {
	defs := make([]string, 0, len(columns))
	for _, col := range columns {
		colType := col.GetType()
		if generic || colType == "" {
			colType = genericColumnType
		}
		defs = append(defs, backtickIdentifier(col.GetName())+" "+colType)
	}
	ddl := fmt.Sprintf("CREATE TABLE %s (%s);", backtickIdentifier(name), strings.Join(defs, ", "))
	return execOK(cat, ddl)
}

// defineView installs a view in the catalog. Unlike tables, the only structured
// input is the view's definition SQL, whose exact shape varies (full
// "CREATE VIEW ..." vs. a bare SELECT). We try the definition as-is, then
// wrapped in CREATE VIEW, and finally fall back to a trivial view so the view
// name still surfaces as a candidate (matching mysql, which reads view names
// straight from metadata) even when the definition cannot be parsed.
func defineView(cat *catalog.Catalog, name, definition string) {
	var attempts []string
	if definition != "" {
		attempts = append(attempts,
			definition,
			fmt.Sprintf("CREATE VIEW %s AS %s", backtickIdentifier(name), definition),
		)
	}
	attempts = append(attempts, fmt.Sprintf("CREATE VIEW %s AS SELECT 1", backtickIdentifier(name)))
	for _, ddl := range attempts {
		if execOK(cat, ddl) {
			return
		}
	}
}

// execOK runs DDL against the catalog and reports whether every statement
// applied without error.
func execOK(cat *catalog.Catalog, ddl string) bool {
	results, err := cat.Exec(ddl, &catalog.ExecOptions{ContinueOnError: true})
	if err != nil {
		return false
	}
	for _, r := range results {
		if r.Error != nil {
			return false
		}
	}
	return true
}

// backtickIdentifier quotes an identifier for TiDB DDL, escaping embedded
// backticks by doubling them.
func backtickIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// isObjectIdentifierCandidate reports whether a candidate type names a schema
// object referenced as an identifier in SQL (and so must be backtick-quoted
// when it is a reserved word or a non-bare name). Keywords, builtin functions,
// and value-like candidates (charset/engine/variable/type) are excluded.
func isObjectIdentifierCandidate(t base.CandidateType) bool {
	switch t {
	case base.CandidateTypeDatabase,
		base.CandidateTypeTable,
		base.CandidateTypeView,
		base.CandidateTypeColumn,
		base.CandidateTypeIndex,
		base.CandidateTypeTrigger,
		base.CandidateTypeEvent,
		base.CandidateTypeRoutine:
		return true
	default:
		return false
	}
}

// quoteIdentifierIfNeeded backtick-quotes an object identifier when it would
// otherwise be invalid as a bare identifier, so that accepting the completion
// inserts valid SQL. When the caret already sits inside a backtick-quoted
// identifier the user is typing, the name is returned unquoted (the user's
// backticks wrap it).
func quoteIdentifierIfNeeded(name string, caretInBacktick bool) string {
	if caretInBacktick || !identifierNeedsQuoting(name) {
		return name
	}
	return backtickIdentifier(name)
}

// identifierNeedsQuoting reports whether name must be backtick-quoted to be a
// valid bare identifier: it is empty, contains a character outside
// [letter,digit,_,$], starts with a digit, or is a reserved keyword.
func identifierNeedsQuoting(name string) bool {
	if name == "" {
		return true
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' {
			return true
		}
	}
	if unicode.IsDigit(rune(name[0])) {
		return true
	}
	return isReservedKeyword(name)
}

// isReservedKeyword reports whether a bare-shaped name is a reserved keyword
// that cannot be used as an unquoted identifier. omni exposes no reserved-word
// predicate, so we parse-check only names that lex as a single keyword token
// (plain identifiers are never reserved), which keeps this cheap for ordinary
// names. A keyword is reserved iff it cannot stand as a bare table reference.
func isReservedKeyword(name string) bool {
	upper := strings.ToUpper(name)
	toks := tidbparser.Tokenize(upper)
	if len(toks) != 1 || tidbparser.TokenName(toks[0].Type) == "" {
		return false
	}
	_, err := tidbparser.Parse("SELECT 1 FROM " + upper)
	return err != nil
}

// caretInsideBacktickIdentifier reports whether the caret sits inside an open
// backtick-quoted identifier (an odd number of backticks precede it), in which
// case completed identifiers must not add their own quotes.
func caretInsideBacktickIdentifier(statement string, pos int) bool {
	if pos > len(statement) {
		pos = len(statement)
	}
	count := 0
	for i := 0; i < pos; i++ {
		if statement[i] == '`' {
			count++
		}
	}
	return count%2 == 1
}

// lineOffsetToBytePos converts a 1-based line number and 0-based character
// (rune) offset to a 0-based byte position in the string.
func lineOffsetToBytePos(s string, line, offset int) int {
	pos := 0
	currentLine := 1
	for pos < len(s) && currentLine < line {
		if s[pos] == '\n' {
			currentLine++
		}
		pos++
	}
	for i := 0; i < offset && pos < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[pos:])
		pos += size
	}
	if pos > len(s) {
		return len(s)
	}
	return pos
}

// omniCandidateTypeToBase maps omni TiDB completion candidate types to the
// Bytebase base types. The six core types (keyword/database/table/view/column/
// function) map to the same base types the mysql completer emits so candidate
// sets are directly comparable.
func omniCandidateTypeToBase(t omnicompletion.CandidateType) base.CandidateType {
	switch t {
	case omnicompletion.CandidateKeyword:
		return base.CandidateTypeKeyword
	case omnicompletion.CandidateDatabase:
		return base.CandidateTypeDatabase
	case omnicompletion.CandidateTable:
		return base.CandidateTypeTable
	case omnicompletion.CandidateView:
		return base.CandidateTypeView
	case omnicompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case omnicompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case omnicompletion.CandidateProcedure:
		return base.CandidateTypeRoutine
	case omnicompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case omnicompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	case omnicompletion.CandidateEvent:
		return base.CandidateTypeEvent
	case omnicompletion.CandidateCharset:
		return base.CandidateTypeCharset
	case omnicompletion.CandidateEngine:
		return base.CandidateTypeEngine
	case omnicompletion.CandidateVariable:
		return base.CandidateTypeSystemVar
	case omnicompletion.CandidateType_:
		return base.CandidateTypeKeyword
	default:
		return base.CandidateTypeNone
	}
}
