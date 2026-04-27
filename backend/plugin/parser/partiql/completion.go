package partiql

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	omnipartiql "github.com/bytebase/omni/partiql"
	"github.com/bytebase/omni/partiql/catalog"
	"github.com/bytebase/omni/partiql/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_DYNAMODB, Completion)
}

// Completion provides auto-complete candidates for PartiQL statements.
// It uses the omni completion engine for keyword and table suggestions,
// and supplements with column suggestions from bytebase metadata.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	// Build omni catalog from bytebase metadata.
	cat := catalog.New()
	var databaseMetadata *model.DatabaseMetadata
	if cCtx.Metadata != nil && cCtx.DefaultDatabase != "" {
		_, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, cCtx.DefaultDatabase)
		if err == nil && dbMeta != nil {
			databaseMetadata = dbMeta
			schema := dbMeta.GetSchemaMetadata("")
			if schema != nil {
				for _, table := range schema.ListTableNames() {
					cat.AddTable(table)
				}
			}
		}
	}

	// Convert (line, offset) to byte position in the statement.
	pos := lineOffsetToBytePos(statement, caretLine, caretOffset)

	// Get omni completion candidates.
	omniCandidates := completion.Complete(statement, pos, cat)

	// Convert to base.Candidate format.
	candidateMap := make(map[string]base.Candidate)
	for _, c := range omniCandidates {
		var candidateType base.CandidateType
		switch c.Kind {
		case "keyword":
			candidateType = base.CandidateTypeKeyword
		case "table":
			candidateType = base.CandidateTypeTable
		default:
			candidateType = base.CandidateTypeNone
		}
		key := candidateKey(c.Text, candidateType)
		candidateMap[key] = base.Candidate{
			Type: candidateType,
			Text: c.Text,
		}
	}

	// Add column completions if we detect a SELECT-item context.
	// The omni completion engine handles keyword/table contexts; for columns,
	// we check whether the cursor is in a position where columns are relevant
	// (after SELECT but before FROM, or in WHERE/HAVING clauses).
	if databaseMetadata != nil && isColumnContext(statement, pos) {
		addColumnCandidates(candidateMap, statement, pos, databaseMetadata)
	}

	// Filter candidates by scene. In query-only mode, exclude DML/DDL
	// keywords that are not valid in read-only editor contexts.
	if cCtx.Scene == base.SceneTypeQuery {
		for key, c := range candidateMap {
			if c.Type == base.CandidateTypeKeyword && isDMLKeyword(c.Text) {
				delete(candidateMap, key)
			}
		}
	}

	// Sort and return.
	var result []base.Candidate
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

// lineOffsetToBytePos converts a 1-based line number and 0-based
// character (rune) offset to a 0-based byte position in the string.
// The offset is in characters, not bytes, so we iterate runes to
// handle multibyte UTF-8 correctly.
func lineOffsetToBytePos(s string, line, offset int) int {
	pos := 0
	currentLine := 1
	for pos < len(s) && currentLine < line {
		if s[pos] == '\n' {
			currentLine++
		}
		pos++
	}
	// Now advance by `offset` runes (not bytes) within the target line.
	for i := 0; i < offset && pos < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[pos:])
		pos += size
	}
	if pos > len(s) {
		return len(s)
	}
	return pos
}

// candidateKey produces a dedup key for a candidate.
func candidateKey(text string, typ base.CandidateType) string {
	return string(typ) + ":" + text
}

// isColumnContext returns true if the cursor position appears to be in a
// context where column names should be suggested. This covers:
//   - Between SELECT and FROM (the projection list)
//   - After FROM when inside a clause that takes expressions: WHERE,
//     HAVING, SET, ORDER BY, GROUP BY, and any operator/function position
//     within those clauses (e.g., WHERE Artist = |)
//
// Table-only contexts (immediately after FROM/JOIN/INTO) return false.
func isColumnContext(statement string, pos int) bool {
	if pos > len(statement) {
		pos = len(statement)
	}
	// Strip quoted identifiers before uppercasing so that "order" doesn't
	// match the ORDER keyword in lastKeywordIndex.
	before := strings.ToUpper(stripQuotedIdentifiers(statement[:pos]))

	// Strip any partial identifier the user is currently typing.
	trimmed := strings.TrimRight(before, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$\"")
	trimmed = strings.TrimRight(trimmed, " \t\n\r")

	// Table context: cursor is right after FROM/JOIN/INTO.
	if strings.HasSuffix(trimmed, "FROM") ||
		strings.HasSuffix(trimmed, "JOIN") ||
		strings.HasSuffix(trimmed, "INTO") {
		return false
	}

	// Between SELECT and FROM (the projection list) — column context.
	selectIdx := lastKeywordIndex(before, "SELECT")
	fromIdx := lastKeywordIndex(before, "FROM")
	if selectIdx >= 0 && (fromIdx < 0 || fromIdx < selectIdx) {
		return true
	}

	// After FROM: if any expression-bearing clause keyword appears in the
	// text before the cursor, we're in a column context.
	for _, kw := range []string{"WHERE", "HAVING", "SET", "ORDER", "GROUP", "BY", "AND", "OR", "ON", "WHEN", "THEN", "ELSE"} {
		kwIdx := lastKeywordIndex(before, kw)
		if kwIdx >= 0 && kwIdx > fromIdx {
			return true
		}
	}

	// Fallback: if FROM was seen and the cursor is past it in a position
	// that is NOT right after FROM/JOIN/INTO (already excluded above),
	// we're likely in an expression context within a clause.
	if fromIdx >= 0 {
		for _, kw := range []string{"WHERE", "HAVING", "SET", "ORDER", "GROUP"} {
			if lastKeywordIndex(before, kw) > fromIdx {
				return true
			}
		}
	}

	return false
}

// lastKeywordIndex returns the index of the last occurrence of kw in s
// that appears at a word boundary (not inside an identifier). Returns -1
// if not found. Both s and kw must be uppercase.
func lastKeywordIndex(s, kw string) int {
	search := s
	for {
		idx := strings.LastIndex(search, kw)
		if idx < 0 {
			return -1
		}
		end := idx + len(kw)
		startOk := idx == 0 || !isIdentRune(rune(search[idx-1]))
		endOk := end >= len(search) || !isIdentRune(rune(search[end]))
		if startOk && endOk {
			return idx
		}
		// Shrink the search window and try again.
		search = search[:idx]
	}
}

func isIdentRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '$'
}

// addColumnCandidates extracts column names from tables referenced in
// the active statement (the one containing the cursor) and adds them
// to the candidate map. It isolates the current statement by finding
// the semicolon boundaries around pos, then scans the FULL current
// statement (including text after the cursor, e.g., FROM clauses that
// follow a SELECT |).
func addColumnCandidates(m map[string]base.Candidate, statement string, pos int, dbMeta *model.DatabaseMetadata) {
	schema := dbMeta.GetSchemaMetadata("")
	if schema == nil {
		return
	}

	// Isolate the current statement: find the semicolons before and after pos.
	currentStmt := extractCurrentStatement(statement, pos)
	tables := extractTableNamesFromStatement(currentStmt)
	if len(tables) == 0 {
		// If no FROM clause found, try all tables.
		tables = schema.ListTableNames()
	}

	for _, tableName := range tables {
		// Case-sensitive lookup to match DynamoDB behavior.
		var actualName string
		for _, t := range schema.ListTableNames() {
			if strings.EqualFold(t, tableName) {
				actualName = t
				break
			}
		}
		if actualName == "" {
			continue
		}
		table := schema.GetTable(actualName)
		if table == nil {
			continue
		}
		for _, col := range table.GetProto().GetColumns() {
			key := candidateKey(col.Name, base.CandidateTypeColumn)
			if _, exists := m[key]; !exists {
				m[key] = base.Candidate{
					Type: base.CandidateTypeColumn,
					Text: col.Name,
				}
			}
		}
	}
}

// extractTableNamesFromStatement scans the statement to find table names
// after FROM/JOIN/INTO keywords. Handles:
//   - Comma-separated tables with or without spaces: FROM Music, Album / FROM Music,Album
//   - Quoted identifiers that match keywords: FROM "order" → table "order" (not keyword ORDER)
//   - Aliases: FROM Music AS m, Album a → extracts Music and Album
//   - Comments between keywords and table names (stripped before scanning)
func extractTableNamesFromStatement(statement string) []string {
	// Strip comments, then split on whitespace AND commas to handle
	// both "Music, Album" and "Music,Album" uniformly.
	cleaned := stripSQLComments(statement)
	tokens := tokenizeForTableExtraction(cleaned)

	var tables []string
	i := 0
	for i < len(tokens) {
		upper := strings.ToUpper(tokens[i].text)
		if (upper == "FROM" || upper == "JOIN" || upper == "INTO") && i+1 < len(tokens) {
			i++
			// Collect table names from a comma-separated list.
			for i < len(tokens) {
				tok := tokens[i]
				// A quoted identifier is always a table name, even if
				// its unquoted form matches a keyword.
				name := tok.text
				if tok.quoted {
					tables = append(tables, name)
				} else if isKeyword(strings.ToUpper(name)) {
					break
				} else {
					tables = append(tables, name)
				}
				i++
				// Skip optional AS + alias.
				if i < len(tokens) && strings.ToUpper(tokens[i].text) == "AS" {
					i += 2 // skip AS and alias
				} else if i < len(tokens) && !tokens[i].comma && !isKeyword(strings.ToUpper(tokens[i].text)) {
					// Implicit alias — skip one token.
					i++
				}
				// If the next token is a comma separator, continue.
				if i < len(tokens) && tokens[i].comma {
					i++ // skip the comma marker
					continue
				}
				break
			}
		} else {
			i++
		}
	}
	return tables
}

// tableToken is a token in the table-extraction scanner.
type tableToken struct {
	text   string // the identifier or keyword text (quotes stripped)
	quoted bool   // true if the original was double-quoted
	comma  bool   // true if this is a comma separator
}

// tokenizeForTableExtraction splits a comment-stripped SQL string into
// tokens suitable for table-name extraction. It splits on whitespace
// and commas, handling:
//   - "Music,Album" → [Music] [,] [Album]
//   - "Music , Album" → [Music] [,] [Album]
//   - `"order"` → [{text:"order", quoted:true}]
func tokenizeForTableExtraction(s string) []tableToken {
	var tokens []tableToken
	i := 0
	for i < len(s) {
		// Skip whitespace.
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			i++
			continue
		}
		// Comma is its own token.
		if s[i] == ',' {
			tokens = append(tokens, tableToken{comma: true})
			i++
			continue
		}
		// Double-quoted identifier.
		if s[i] == '"' {
			i++ // skip opening "
			start := i
			for i < len(s) && s[i] != '"' {
				if s[i] == '"' && i+1 < len(s) && s[i+1] == '"' {
					i += 2
					continue
				}
				i++
			}
			text := s[start:i]
			if i < len(s) {
				i++ // skip closing "
			}
			tokens = append(tokens, tableToken{text: text, quoted: true})
			continue
		}
		// Bare word: identifier or keyword.
		start := i
		for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' &&
			s[i] != '\r' && s[i] != ',' && s[i] != '"' && s[i] != ';' &&
			s[i] != '(' && s[i] != ')' {
			i++
		}
		if i > start {
			tokens = append(tokens, tableToken{text: s[start:i]})
		} else {
			// Delimiter character like (, ), ; — skip it to avoid
			// an infinite loop.
			i++
		}
	}
	return tokens
}

// stripSQLComments removes line comments (--...) and block comments
// (/*...*/) from a SQL string, replacing them with spaces to preserve
// token boundaries. Respects single-quoted strings.
func stripSQLComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		switch {
		case s[i] == '\'':
			// Single-quoted string: copy verbatim.
			b.WriteByte(s[i])
			i++
			for i < len(s) {
				b.WriteByte(s[i])
				if s[i] == '\'' {
					i++
					if i < len(s) && s[i] == '\'' {
						b.WriteByte(s[i])
						i++
						continue
					}
					break
				}
				i++
			}
		case s[i] == '-' && i+1 < len(s) && s[i+1] == '-':
			// Line comment: replace with space.
			b.WriteByte(' ')
			i += 2
			for i < len(s) && s[i] != '\n' && s[i] != '\r' {
				i++
			}
		case s[i] == '/' && i+1 < len(s) && s[i+1] == '*':
			// Block comment: replace with space.
			b.WriteByte(' ')
			i += 2
			depth := 1
			for i+1 < len(s) && depth > 0 {
				if s[i] == '/' && s[i+1] == '*' {
					depth++
					i += 2
				} else if s[i] == '*' && s[i+1] == '/' {
					depth--
					i += 2
				} else {
					i++
				}
			}
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// extractCurrentStatement returns the statement segment containing pos
// by using omni's Split to find the correct semicolon boundaries. This
// respects strings, comments, and Ion literals — a semicolon inside
// 'a;b' won't split the statement.
func extractCurrentStatement(statement string, pos int) string {
	segs := omnipartiql.Split(statement)
	for _, seg := range segs {
		if pos >= seg.ByteStart && pos <= seg.ByteEnd {
			return seg.Text
		}
	}
	// Fallback: return the whole statement.
	return statement
}

// isKeyword returns true if the given uppercase word is a SQL keyword.
func isKeyword(w string) bool {
	switch w {
	case "SELECT", "FROM", "WHERE", "AND", "OR", "NOT", "INSERT", "INTO",
		"VALUE", "VALUES", "UPDATE", "SET", "DELETE", "CREATE", "DROP",
		"TABLE", "INDEX", "AS", "AT", "BY", "ON", "IN", "IS", "LIKE",
		"BETWEEN", "ORDER", "GROUP", "HAVING", "LIMIT", "OFFSET",
		"JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "CROSS",
		"UNION", "INTERSECT", "EXCEPT", "DISTINCT", "ALL",
		"NULL", "MISSING", "TRUE", "FALSE", "CAST", "CASE", "WHEN",
		"THEN", "ELSE", "END", "EXISTS", "ASC", "DESC", "EXPLAIN":
		return true
	}
	return false
}

// isDMLKeyword returns true if the keyword is a DML/DDL operation that
// should be excluded in query-only (read-only) editor scenes.
func isDMLKeyword(w string) bool {
	switch strings.ToUpper(w) {
	case "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "REPLACE",
		"UPSERT", "REMOVE", "SET":
		return true
	}
	return false
}

// stripQuotedIdentifiers replaces double-quoted identifiers with
// placeholder text so that keyword searches don't match content inside
// quotes (e.g., "order" should not be treated as the ORDER keyword).
func stripQuotedIdentifiers(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '"' {
			// Replace the entire "..." with underscores to preserve length.
			b.WriteByte('_')
			i++ // skip opening "
			for i < len(s) {
				if s[i] == '"' {
					i++
					if i < len(s) && s[i] == '"' {
						b.WriteByte('_')
						b.WriteByte('_')
						i++
						continue
					}
					b.WriteByte('_')
					break
				}
				b.WriteByte('_')
				i++
			}
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}
