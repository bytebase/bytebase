package tidb

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bytebase/omni/tidb/catalog"
	omnicompletion "github.com/bytebase/omni/tidb/completion"

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
	cat := buildCatalog(ctx, cCtx)
	pos := lineOffsetToBytePos(statement, caretLine, caretOffset)

	candidateMap := make(map[string]base.Candidate)
	for _, c := range omnicompletion.Complete(statement, pos, cat) {
		t := omniCandidateTypeToBase(c.Type)
		candidateMap[string(t)+":"+c.Text] = base.Candidate{
			Type:       t,
			Text:       c.Text,
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
// replaying minimal DDL. Every identifier is backticked so reserved words and
// special characters parse correctly; tables are created one at a time so a
// single unparseable column type cannot empty the whole catalog.
func buildCatalog(ctx context.Context, cCtx base.CompletionContext) *catalog.Catalog {
	cat := catalog.New()
	if cCtx.Metadata == nil || cCtx.DefaultDatabase == "" {
		return cat
	}
	_, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, cCtx.DefaultDatabase)
	if err != nil || dbMeta == nil {
		return cat
	}

	db := backtickIdentifier(cCtx.DefaultDatabase)
	if _, err := cat.Exec(fmt.Sprintf("CREATE DATABASE %s; USE %s;", db, db), &catalog.ExecOptions{ContinueOnError: true}); err != nil {
		return cat
	}

	schema := dbMeta.GetSchemaMetadata("")
	if schema == nil {
		return cat
	}
	for _, tableName := range schema.ListTableNames() {
		table := schema.GetTable(tableName)
		if table == nil {
			continue
		}
		defineTable(cat, tableName, table.GetProto().GetColumns())
	}
	for _, viewName := range schema.ListViewNames() {
		view := schema.GetView(viewName)
		if view == nil {
			continue
		}
		defineView(cat, viewName, view.GetDefinition())
	}
	return cat
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
