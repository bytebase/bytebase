package base

import (
	"context"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
)

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

// CompleteMySQLFamily runs auto-completion for a MySQL-family engine (MySQL,
// MariaDB, OceanBase, TiDB) using the dialect's statement splitter, reserved-word
// predicate, and omni completion callback.
func CompleteMySQLFamily(
	cCtx CompletionContext,
	statement string,
	caretLine int,
	caretOffset int,
	currentStatement func(statement string, pos int) (string, int),
	isReservedKeyword func(name string) bool,
	complete func(stmt string, pos int) []Candidate,
) []Candidate {
	// Limit completion to the statement containing the caret so table refs from
	// earlier statements in the buffer don't leak into the candidate set.
	stmt, pos := currentStatement(statement, lineOffsetToBytePos(statement, caretLine, caretOffset))

	caretInBacktick := CaretInsideMySQLBacktickIdentifier(stmt, pos)
	candidateMap := make(map[string]Candidate)
	for _, c := range complete(stmt, pos) {
		t := c.Type
		text := c.Text
		switch {
		case isObjectIdentifierCandidate(t):
			text = QuoteMySQLIdentifierIfNeeded(c.Text, caretInBacktick, isReservedKeyword)
		case t == CandidateTypeFunction:
			text = c.Text + "()"
		default:
			// Keywords and value-like candidates pass through unchanged.
		}
		candidateMap[string(t)+":"+text] = Candidate{
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
	if cCtx.Scene == SceneTypeQuery && caretAtStatementStart(stmt, pos) {
		for key, c := range candidateMap {
			if c.Type == CandidateTypeKeyword && writeStatementKeywords[strings.ToUpper(c.Text)] {
				delete(candidateMap, key)
			}
		}
	}

	result := make([]Candidate, 0, len(candidateMap))
	for _, c := range candidateMap {
		result = append(result, c)
	}
	slices.SortFunc(result, func(a, b Candidate) int {
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
	return result
}

// PopulateMySQLFamilyCatalog populates an omni MySQL-family catalog from
// Bytebase metadata. It loads the default database plus any database referenced
// as a qualifier in the statement (so cross-database qualified completion like
// `other_db.tbl.col` resolves). Unreferenced database names are enumerated and
// registered only when the cursor position can suggest database-name candidates.
func PopulateMySQLFamilyCatalog(
	ctx context.Context,
	cCtx CompletionContext,
	statement string,
	pos int,
	hasDatabaseRule func(sql string, offset int) bool,
	registerDB func(name string),
	loadDB func(name string, meta *metadatapb.DatabaseSchemaMetadata),
	setCurrentDB func(name string),
) {
	suggestsDB := hasDatabaseRule != nil && suggestsMySQLDatabaseRef(statement, pos, hasDatabaseRule)
	qualifiers := databaseQualifiersInStatement(statement, cCtx.DefaultDatabase)

	var allNames []string
	if (suggestsDB || len(qualifiers) > 0) && cCtx.ListDatabaseNames != nil {
		if names, err := cCtx.ListDatabaseNames(ctx, cCtx.InstanceID); err == nil {
			allNames = names
		}
	}

	seen := map[string]bool{}
	var loadOrder []string
	if cCtx.Metadata != nil {
		if cCtx.DefaultDatabase != "" {
			loadOrder = append(loadOrder, cCtx.DefaultDatabase)
			seen[cCtx.DefaultDatabase] = true
		}
		if len(qualifiers) > 0 {
			for _, name := range allNames {
				if !seen[name] && qualifiers[strings.ToLower(name)] {
					loadOrder = append(loadOrder, name)
					seen[name] = true
				}
			}
		}
	}

	// Register unreferenced database names (name only) only when the completion
	// position can actually suggest database_ref candidates.
	if suggestsDB {
		for _, name := range allNames {
			if !seen[name] {
				registerDB(name)
			}
		}
	}

	for _, name := range loadOrder {
		var proto *metadatapb.DatabaseSchemaMetadata
		if _, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, name); err == nil && dbMeta != nil {
			proto = dbMeta.GetProto()
		}
		if proto == nil {
			registerDB(name)
			continue
		}
		loadDB(name, proto)
	}

	setCurrentDB(cCtx.DefaultDatabase)
}

// suggestsMySQLDatabaseRef reports whether the parser's C3 collector at pos
// (including the placeholder fallback strategies omni uses) requests the
// "database_ref" rule.
func suggestsMySQLDatabaseRef(stmt string, pos int, hasDatabaseRule func(sql string, offset int) bool) bool {
	if pos > len(stmt) {
		pos = len(stmt)
	}
	collectOffset := pos - extractIdentPrefixLen(stmt, pos)
	if hasDatabaseRule(stmt, collectOffset) {
		return true
	}
	prefix := stmt[:collectOffset]
	suffix := stmt[collectOffset:]
	for _, patched := range []string{
		prefix + " __placeholder__" + suffix,
		prefix + " __placeholder__ " + suffix,
		prefix + " 1" + suffix,
	} {
		if hasDatabaseRule(patched, collectOffset) {
			return true
		}
	}
	return false
}

func extractIdentPrefixLen(sql string, cursorOffset int) int {
	i := cursorOffset
	for i > 0 {
		c := sql[i-1]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_' {
			i--
		} else {
			break
		}
	}
	return cursorOffset - i
}

// databaseQualifiersInStatement returns the lower-cased identifiers that appear
// immediately before a '.' qualifier in statement (excluding defaultDB).
func databaseQualifiersInStatement(statement, defaultDB string) map[string]bool {
	s := stripStringsAndComments(statement)
	if !strings.Contains(s, ".") {
		return nil
	}
	defaultLower := strings.ToLower(defaultDB)
	var out map[string]bool
	for i := 0; i < len(s); i++ {
		if s[i] != '.' {
			continue
		}
		j := i
		for j > 0 && isSpaceByte(s[j-1]) {
			j--
		}
		if j == 0 {
			continue
		}
		var name string
		if s[j-1] == '`' {
			end := j - 1
			k := end - 1
			for k >= 0 {
				if s[k] == '`' {
					if k > 0 && s[k-1] == '`' {
						k -= 2
						continue
					}
					break
				}
				k--
			}
			if k >= 0 {
				name = strings.ReplaceAll(s[k+1:end], "``", "`")
			}
		} else if isIdentByte(s[j-1]) {
			end := j
			k := end - 1
			for k > 0 && isIdentByte(s[k-1]) {
				k--
			}
			name = s[k:end]
		}
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		if lower == defaultLower {
			continue
		}
		if out == nil {
			out = make(map[string]bool)
		}
		out[lower] = true
	}
	return out
}

// MySQLStatementReferencesDatabase reports whether dbName appears as a database
// qualifier (`dbName.`) in the statement, at an identifier boundary.
func MySQLStatementReferencesDatabase(statement, dbName string) bool {
	if dbName == "" {
		return false
	}
	s := strings.ToLower(stripStringsAndComments(statement))
	name := strings.ToLower(dbName)
	for _, q := range []string{name, "`" + name + "`"} {
		from := 0
		for {
			i := strings.Index(s[from:], q)
			if i < 0 {
				break
			}
			at := from + i
			from = at + 1
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

func backtickIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func isObjectIdentifierCandidate(t CandidateType) bool {
	switch t {
	case CandidateTypeDatabase,
		CandidateTypeTable,
		CandidateTypeView,
		CandidateTypeColumn,
		CandidateTypeIndex,
		CandidateTypeTrigger,
		CandidateTypeEvent,
		CandidateTypeRoutine:
		return true
	default:
		return false
	}
}

// QuoteMySQLIdentifierIfNeeded backtick-quotes an object identifier when it
// would otherwise be invalid as a bare identifier, so that accepting the
// completion inserts valid SQL. When the caret already sits inside a
// backtick-quoted identifier the user is typing, the name is returned unquoted.
func QuoteMySQLIdentifierIfNeeded(name string, caretInBacktick bool, isReservedKeyword func(string) bool) string {
	if caretInBacktick || !identifierNeedsQuoting(name, isReservedKeyword) {
		return name
	}
	return backtickIdentifier(name)
}

func identifierNeedsQuoting(name string, isReservedKeyword func(string) bool) bool {
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
	return isReservedKeyword != nil && isReservedKeyword(name)
}

const identifierChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$"

func caretAtStatementStart(statement string, pos int) bool {
	if pos > len(statement) {
		pos = len(statement)
	}
	before := stripStringsAndComments(statement)[:pos]
	before = strings.TrimRight(before, identifierChars)
	before = strings.TrimRight(before, " \t\r\n")
	return before == "" || before[len(before)-1] == ';'
}

// CaretInsideMySQLBacktickIdentifier reports whether the caret sits inside an
// open backtick-quoted identifier (an odd number of backticks precede it).
func CaretInsideMySQLBacktickIdentifier(statement string, pos int) bool {
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
// Backtick-quoted identifiers are left intact.
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
				if out[i] == '\\' && i+1 < n {
					blank(i)
					blank(i + 1)
					i += 2
					continue
				}
				if out[i] == q {
					if i+1 < n && out[i+1] == q {
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
			i++
			for i < n {
				if out[i] == '`' {
					if i+1 < n && out[i+1] == '`' {
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
