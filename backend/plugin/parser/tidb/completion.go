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
	// Limit completion to the statement containing the caret so table refs from
	// earlier statements in the buffer don't leak into the candidate set.
	stmt, pos := currentStatement(statement, lineOffsetToBytePos(statement, caretLine, caretOffset))

	cat := buildCatalog(ctx, cCtx, stmt)
	caretInBacktick := caretInsideBacktickIdentifier(stmt, pos)
	candidateMap := make(map[string]base.Candidate)
	for _, c := range omnicompletion.Complete(stmt, pos, cat) {
		t := omniCandidateTypeToBase(c.Type)
		text := c.Text
		switch {
		case isObjectIdentifierCandidate(t):
			text = quoteIdentifierIfNeeded(c.Text, caretInBacktick)
		case t == base.CandidateTypeFunction:
			text = c.Text + "()"
		default:
			// Keywords and value-like candidates pass through unchanged.
		}
		candidateMap[string(t)+":"+text] = base.Candidate{
			Type:       t,
			Text:       text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}
	}

	// In the read-only query scene, drop keywords that *initiate* writes
	// (DML/DDL/transaction/admin statements) so the editor doesn't offer to start
	// a write statement. This only applies at a statement-start position: the same
	// keywords are valid sub-keywords inside read statements (e.g. CREATE in SHOW
	// CREATE TABLE, UPDATE in SELECT ... FOR UPDATE) and must be preserved there.
	if cCtx.Scene == base.SceneTypeQuery && caretAtStatementStart(stmt, pos) {
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
	allNames := listAllDatabaseNames(ctx, cCtx)

	// Decide which databases to fully load with their objects. This needs a
	// metadata fetcher and a database to load from: the default database (if one
	// is selected) plus any database referenced as a qualifier in the statement.
	seen := map[string]bool{}
	var loadOrder []string
	if cCtx.Metadata != nil {
		if cCtx.DefaultDatabase != "" {
			loadOrder = append(loadOrder, cCtx.DefaultDatabase)
			seen[cCtx.DefaultDatabase] = true
		}
		for _, name := range allNames {
			if !seen[name] && statementReferencesDatabase(statement, name) {
				loadOrder = append(loadOrder, name)
				seen[name] = true
			}
		}
	}

	// Register every other known database name (name only) so database-name
	// candidates surface — this works at the instance level even when no database
	// is selected (DefaultDatabase empty).
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
	if cCtx.DefaultDatabase != "" {
		_, _ = cat.Exec("USE "+backtickIdentifier(cCtx.DefaultDatabase)+";", &catalog.ExecOptions{ContinueOnError: true})
	}
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
	s := strings.ToLower(stripStringsAndComments(statement))
	name := strings.ToLower(dbName)
	// Match the database name (bare or backtick-quoted) at an identifier
	// boundary, followed by optional whitespace and a dot. TiDB allows whitespace
	// around the qualifier dot (e.g. `db . table`).
	for _, q := range []string{name, "`" + name + "`"} {
		from := 0
		for {
			i := strings.Index(s[from:], q)
			if i < 0 {
				break
			}
			at := from + i
			from = at + 1
			// The bare form must start at an identifier boundary so `mydb`
			// doesn't match inside `notmydb`.
			if q[0] != '`' && at > 0 && isIdentByte(s[at-1]) {
				continue
			}
			j := at + len(q)
			for j < len(s) && isSpaceByte(s[j]) {
				j++
			}
			if j < len(s) && s[j] == '.' {
				return true
			}
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

// execOK runs DDL against the catalog and reports whether it actually installed
// a schema object: every statement parsed without error AND at least one was a
// non-DML (utility) statement that mutated the catalog. A definition that
// reduces to only skipped DML — e.g. a bare SELECT view body, which TiDB sync
// stores in information_schema.VIEWS.VIEW_DEFINITION — is NOT a success, so the
// caller falls through to the wrapped CREATE VIEW form.
func execOK(cat *catalog.Catalog, ddl string) bool {
	results, err := cat.Exec(ddl, &catalog.ExecOptions{ContinueOnError: true})
	if err != nil {
		return false
	}
	applied := false
	for _, r := range results {
		if r.Error != nil {
			return false
		}
		if !r.Skipped {
			applied = true
		}
	}
	return applied
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
// identifierChars is the set of bytes that can appear in a bare identifier,
// used to strip a partial identifier the user is typing.
const identifierChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$"

// caretAtStatementStart reports whether the caret is at a statement-start
// position — only whitespace and comments precede it (or it directly follows a
// ';'), ignoring any partial identifier the user is currently typing (so "CRE"
// or "...; INS" still count as statement start). Write keywords are filtered
// only here (in the query scene) so keywords valid inside read statements are
// preserved in other positions.
func caretAtStatementStart(statement string, pos int) bool {
	if pos > len(statement) {
		pos = len(statement)
	}
	before := stripStringsAndComments(statement)[:pos]
	before = strings.TrimRight(before, identifierChars) // drop the partial identifier
	before = strings.TrimRight(before, " \t\r\n")
	return before == "" || before[len(before)-1] == ';'
}

func caretInsideBacktickIdentifier(statement string, pos int) bool {
	if pos > len(statement) {
		pos = len(statement)
	}
	masked := stripStringsAndComments(statement)
	count := 0
	for i := 0; i < pos; i++ {
		if masked[i] == '`' {
			count++
		}
	}
	return count%2 == 1
}

// stripStringsAndComments returns statement with the contents of string literals
// ('...', "..."), line comments (-- , #), and block comments (/* */) replaced by
// spaces, preserving byte length (newlines kept) so caret offsets stay valid.
// Backtick-quoted identifiers are left intact. This lets the lexical heuristics
// above ignore backticks and database names that appear inside literals or
// comments rather than as real SQL tokens.
func stripStringsAndComments(statement string) string {
	out := []byte(statement)
	n := len(out)
	blank := func(i int) {
		if out[i] != '\n' {
			out[i] = ' '
		}
	}
	i := 0
	for i < n {
		switch c := out[i]; {
		case c == '\'' || c == '"':
			q := c
			blank(i)
			i++
			for i < n {
				if out[i] == '\\' && i+1 < n { // backslash escape
					blank(i)
					blank(i + 1)
					i += 2
					continue
				}
				if out[i] == q {
					if i+1 < n && out[i+1] == q { // doubled-quote escape
						blank(i)
						blank(i + 1)
						i += 2
						continue
					}
					blank(i)
					i++
					break
				}
				blank(i)
				i++
			}
		case c == '#':
			for i < n && out[i] != '\n' {
				out[i] = ' '
				i++
			}
		case c == '-' && i+1 < n && out[i+1] == '-' && (i+2 >= n || isSpaceByte(out[i+2])):
			for i < n && out[i] != '\n' {
				out[i] = ' '
				i++
			}
		case c == '/' && i+1 < n && out[i+1] == '*':
			blank(i)
			blank(i + 1)
			i += 2
			for i < n {
				if out[i] == '*' && i+1 < n && out[i+1] == '/' {
					blank(i)
					blank(i + 1)
					i += 2
					break
				}
				blank(i)
				i++
			}
		case c == '`':
			// Leave backtick-quoted identifiers intact; skip to the closing
			// backtick so quote characters inside them are not misread.
			i++
			for i < n {
				if out[i] == '`' {
					if i+1 < n && out[i+1] == '`' { // doubled backtick
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
		default:
			i++
		}
	}
	return string(out)
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
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

// currentStatement returns the statement containing byte position pos within
// statement, along with pos translated to an offset within that statement. It
// uses omni's splitter so completion sees only the caret's statement, not table
// references from earlier statements in the buffer.
func currentStatement(statement string, pos int) (string, int) {
	for _, seg := range tidbparser.Split(statement) {
		if pos >= seg.ByteStart && pos <= seg.ByteEnd {
			return seg.Text, pos - seg.ByteStart
		}
	}
	return statement, pos
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
