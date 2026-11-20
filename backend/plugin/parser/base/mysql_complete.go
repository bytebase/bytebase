package base

import (
	"cmp"
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
	// DML / data loading.
	"INSERT": true, "UPDATE": true, "DELETE": true, "REPLACE": true,
	"LOAD": true, "IMPORT": true, "BATCH": true,
	// DDL.
	"CREATE": true, "ALTER": true, "DROP": true, "TRUNCATE": true, "RENAME": true,
	// Privileges / session.
	"GRANT": true, "REVOKE": true, "SET": true, "USE": true,
	// Transaction control.
	"BEGIN": true, "START": true, "COMMIT": true, "ROLLBACK": true,
	"SAVEPOINT": true, "RELEASE": true, "XA": true,
	// Admin / maintenance / replication.
	"ANALYZE": true, "OPTIMIZE": true, "FLUSH": true, "REPAIR": true,
	"CHECK": true, "CHECKSUM": true, "RESET": true, "KILL": true,
	"LOCK": true, "UNLOCK": true, "CACHE": true,
	"SHUTDOWN": true, "RESTART": true, "CLONE": true,
	"INSTALL": true, "UNINSTALL": true, "CHANGE": true, "STOP": true,
	"PURGE": true, "BINLOG": true, "HANDLER": true,
	"SIGNAL": true, "RESIGNAL": true,
	// TiDB maintenance / backup / resource / region statements.
	"ADMIN": true, "SPLIT": true, "DISTRIBUTE": true, "CALIBRATE": true,
	"QUERY": true, "BACKUP": true, "RESTORE": true, "RECOVER": true,
	"FLASHBACK": true, "PLAN": true,
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
		if t == CandidateTypeNone {
			continue
		}
		text := c.Text
		switch {
		case isObjectIdentifierCandidate(t):
			text = QuoteMySQLIdentifierIfNeeded(c.Text, caretInBacktick, isReservedKeyword)
		case t == CandidateTypeFunction:
			if !strings.HasSuffix(text, "()") {
				text += "()"
			}
		default:
			// Keywords and value-like candidates pass through unchanged.
		}
		candidate := Candidate{
			Type:       t,
			Text:       text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}
		candidateMap[candidate.String()] = candidate
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
		return cmp.Or(
			cmp.Compare(a.Type, b.Type),
			cmp.Compare(a.Text, b.Text),
			cmp.Compare(a.Definition, b.Definition),
			cmp.Compare(a.Comment, b.Comment),
		)
	})
	return result
}

// PopulateMySQLFamilyCatalog populates an omni MySQL-family catalog from
// Bytebase metadata. It loads the default database plus any database referenced
// as a database qualifier in the statement (so cross-database qualified
// completion like `other_db.tbl.col` resolves without enumerating all
// instance databases). Unreferenced database names are enumerated via
// ListDatabaseNames and registered only when the cursor position can suggest
// database-name candidates.
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

	var allNames []string
	enumerated := false
	ensureEnumerated := func() {
		if enumerated || cCtx.ListDatabaseNames == nil {
			return
		}
		enumerated = true
		if names, err := cCtx.ListDatabaseNames(ctx, cCtx.InstanceID); err == nil {
			allNames = names
		}
	}
	if suggestsDB {
		ensureEnumerated()
	}

	findExactFolded := func(name string) (string, bool) {
		ensureEnumerated()
		lower := strings.ToLower(name)
		for _, n := range allNames {
			if strings.ToLower(n) == lower {
				return n, n != name
			}
		}
		return "", false
	}

	// Load the default database first so its metadata determines whether this
	// instance treats database identifiers as case-sensitive or case-folding.
	caseSensitive := true
	defaultLoaded := false
	seen := map[string]bool{}
	keyFor := func(name string) string {
		if caseSensitive {
			return name
		}
		return strings.ToLower(name)
	}

	defaultName := cCtx.DefaultDatabase
	var defaultProto *metadatapb.DatabaseSchemaMetadata
	if cCtx.Metadata != nil && defaultName != "" {
		if resolvedName, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, defaultName); err == nil && dbMeta != nil {
			if resolvedName != "" {
				defaultName = resolvedName
			}
			caseSensitive = dbMeta.GetIsObjectCaseSensitive()
			defaultProto = dbMeta.GetProto()
			defaultLoaded = true
		}
		seen[keyFor(defaultName)] = true
	}

	qualifiers, caretDBs := analyzeMySQLStatementQualifiers(statement, pos, cCtx.DefaultDatabase, caseSensitive)

	// Omni's catalog maps database names case-insensitively (`toLower(name)`).
	// On a case-sensitive instance where multiple case-distinct DBs differ only
	// by case (e.g. `Foo` vs `foo`), prioritize the databases active at the
	// caret (with projected CTE/subquery DBs first) over colliding siblings, and
	// defer defaultDB only when the caret's leading DB is the colliding sibling.
	if caseSensitive && len(caretDBs) > 0 {
		for idx := len(caretDBs) - 1; idx >= 0; idx-- {
			target := caretDBs[idx]
			for i, q := range qualifiers {
				if q == target && i > 0 {
					qualifiers = append([]string{q}, append(qualifiers[:i:i], qualifiers[i+1:]...)...)
					break
				}
			}
		}
	}
	deferDefault := caseSensitive && defaultName != "" && len(caretDBs) > 0 && caretDBs[0] != defaultName && strings.EqualFold(caretDBs[0], defaultName)
	loadDefault := func() {
		if cCtx.Metadata == nil || defaultName == "" {
			return
		}
		if defaultProto != nil {
			loadDB(defaultName, defaultProto)
		} else {
			registerDB(defaultName)
		}
	}
	if !deferDefault {
		loadDefault()
	}

	if cCtx.Metadata != nil {
		for _, q := range qualifiers {
			loadName := q
			if !caseSensitive && enumerated {
				if exact, diffCase := findExactFolded(q); diffCase {
					loadName = exact
				}
			}
			if seen[keyFor(loadName)] {
				continue
			}
			var proto *metadatapb.DatabaseSchemaMetadata
			if resolvedName, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, loadName); err == nil && dbMeta != nil {
				if resolvedName != "" {
					loadName = resolvedName
				}
				proto = dbMeta.GetProto()
			} else if !defaultLoaded || !caseSensitive {
				// Case-folding instances (lower_case_table_names) may store e.g.
				// "db_b" while the query uses "DB_B"; resolve canonically on miss
				// when defaultDB is case-folding or unavailable (including empty).
				if exact, diffCase := findExactFolded(loadName); diffCase {
					if resolvedName, dbMeta, err := cCtx.Metadata(ctx, cCtx.InstanceID, exact); err == nil && dbMeta != nil && !dbMeta.GetIsObjectCaseSensitive() {
						loadName = exact
						if resolvedName != "" {
							loadName = resolvedName
						}
						proto = dbMeta.GetProto()
					}
				}
			}
			seen[keyFor(loadName)] = true
			seen[keyFor(q)] = true
			if proto != nil {
				loadDB(loadName, proto)
			} else if suggestsDB {
				registerDB(loadName)
			}
		}
	}
	if deferDefault {
		loadDefault()
	}

	// Register unreferenced database names (name only) only when the completion
	// position can actually suggest database_ref candidates.
	if suggestsDB {
		for _, name := range allNames {
			if !seen[keyFor(name)] {
				seen[keyFor(name)] = true
				registerDB(name)
			}
		}
	}

	if defaultName != "" {
		setCurrentDB(defaultName)
	}
}

// suggestsMySQLDatabaseRef reports whether the parser's C3 collector at pos
// (including the placeholder fallback strategies omni uses) requests the
// "database_ref" rule.
func suggestsMySQLDatabaseRef(stmt string, pos int, hasDatabaseRule func(sql string, offset int) bool) bool {
	pos = max(0, min(pos, len(stmt)))
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
	end := max(0, min(cursorOffset, len(sql)))
	i := end
	for i > 0 {
		if c := sql[i-1]; c < utf8.RuneSelf {
			if !isIdentByte(c) {
				break
			}
			i--
			continue
		}
		r, size := utf8.DecodeLastRuneInString(sql[:i])
		if r == utf8.RuneError || !isIdentRune(r) {
			break
		}
		i -= size
	}
	return end - i
}

type mysqlQualToken struct {
	text   string
	upper  string // non-empty only for unquoted word tokens
	punct  byte   // non-zero for '.', ',', '(', ')', ';', '*'
	isWord bool
	end    int
}

func scanMySQLQualTokens(statement string) []mysqlQualToken {
	s := stripStringsAndComments(statement)
	var toks []mysqlQualToken
	n := len(s)
	for i := 0; i < n; {
		b := s[i]
		if isSpaceByte(b) {
			i++
			continue
		}
		switch b {
		case '.', ',', '(', ')', ';', '*':
			i++
			toks = append(toks, mysqlQualToken{punct: b, end: i})
		case '`':
			var buf strings.Builder
			i++
			closed := false
			for i < n {
				if s[i] == '`' {
					if i+1 < n && s[i+1] == '`' {
						buf.WriteByte('`')
						i += 2
						continue
					}
					closed = true
					i++
					break
				}
				buf.WriteByte(s[i])
				i++
			}
			if closed && buf.Len() > 0 {
				toks = append(toks, mysqlQualToken{text: buf.String(), isWord: true, end: i})
			}
		default:
			if size, ok := nextIdentRune(s, i); ok {
				start := i
				i += size
				for i < n {
					sz, isIdent := nextIdentRune(s, i)
					if !isIdent {
						break
					}
					i += sz
				}
				word := s[start:i]
				toks = append(toks, mysqlQualToken{
					text:   word,
					upper:  strings.ToUpper(word),
					isWord: true,
					end:    i,
				})
			} else {
				// Emit a neutral separator token for operators/symbols so adjacent
				// dotted expressions (e.g. `ON a. = o.user_id`) never coalesce into
				// a synthetic three-part chain (`a . o .`).
				i += size
				if len(toks) == 0 || toks[len(toks)-1].punct != ' ' {
					toks = append(toks, mysqlQualToken{punct: ' ', end: i})
				} else {
					toks[len(toks)-1].end = i
				}
			}
		}
	}
	return toks
}

type mysqlProjRef struct {
	db    string
	alias string
}

type mysqlQualScope struct {
	aliasScopeID          int
	cteName               string
	stmtStart             bool
	stmtVerb              string
	inSelectList          bool
	pastFromOrWhere       bool
	isSubquery            bool
	inTableClause         bool
	inInsertReplaceTarget bool
	inGrantPrivList       bool
	allowCommaTable       bool
	allowFromAsTable      bool
	afterCreate           bool
	afterRename           bool
	afterDDLOnObj         bool
	prevKeyword           string
}

type mysqlAliasMap struct {
	exact map[string][]string
	lower map[string][]string
}

func databaseQualifiersInStatement(statement, defaultDB string, caseSensitive bool) []string {
	qualifiers, _ := analyzeMySQLStatementQualifiers(statement, 0, defaultDB, caseSensitive)
	return qualifiers
}

func appendUniqueDB(dst []string, dbs ...string) []string {
	for _, db := range dbs {
		if db == "" || slices.Contains(dst, db) {
			continue
		}
		dst = append(dst, db)
	}
	return dst
}

// analyzeMySQLStatementQualifiers extracts database qualifiers referenced in
// statement (excluding defaultDB), distinguishing them from ordinary
// table/alias qualifiers such as `SELECT a. FROM users a`, and resolves the
// databases associated with any qualifier or table/alias active at pos:
//   - Three-part chains (`db.tbl.col` or `db.tbl.`) always identify `db`.
//   - Two-part chains (`db.obj` or `db.`) only identify `db` inside table/object
//     positions (FROM, JOIN, UPDATE, INSERT/REPLACE [INTO], DELETE, USING,
//     DDL object headers, REFERENCES, CALL, CREATE ... LIKE, etc.).
func analyzeMySQLStatementQualifiers(statement string, pos int, defaultDB string, caseSensitive bool) ([]string, []string) {
	if !strings.Contains(statement, ".") {
		return nil, nil
	}
	toks := scanMySQLQualTokens(statement)
	if len(toks) == 0 {
		return nil, nil
	}

	keyFor := func(name string) string {
		if caseSensitive {
			return name
		}
		return strings.ToLower(name)
	}
	defaultKey := keyFor(defaultDB)
	seen := make(map[string]bool)
	var out []string
	addDB := func(name string) {
		if name == "" {
			return
		}
		key := keyFor(name)
		if key == defaultKey || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, name)
	}

	nextAliasScopeID := 1
	aliasesByScope := map[int]*mysqlAliasMap{}
	parentAliasScope := map[int]int{}
	scopeProjRefs := map[int][]mysqlProjRef{}
	scopeFromDBs := map[int][]string{}
	cteByScope := map[int]map[string][]string{}
	recordProjRef := func(scopeID int, ref mysqlProjRef) {
		if ref.db == "" && ref.alias == "" {
			return
		}
		scopeProjRefs[scopeID] = append(scopeProjRefs[scopeID], ref)
	}
	recordScopeFromDB := func(scopeID int, dbs ...string) {
		scopeFromDBs[scopeID] = appendUniqueDB(scopeFromDBs[scopeID], dbs...)
	}
	resolveAliasDBs := func(scopeID int, ident string) ([]string, bool) {
		lower := strings.ToLower(ident)
		for sid, ok := scopeID, true; ok; sid, ok = parentAliasScope[sid] {
			if am := aliasesByScope[sid]; am != nil {
				if dbs, hit := am.exact[ident]; hit {
					return dbs, true
				}
				if dbs, hit := am.lower[lower]; hit {
					return dbs, true
				}
			}
		}
		return nil, false
	}
	finalizeScopeDBs := func(scopeID int) ([]string, []string) {
		var projDBs []string
		for _, ref := range scopeProjRefs[scopeID] {
			if ref.db != "" {
				projDBs = appendUniqueDB(projDBs, ref.db)
				continue
			}
			if dbs, ok := resolveAliasDBs(scopeID, ref.alias); ok {
				projDBs = appendUniqueDB(projDBs, dbs...)
			}
		}
		allDBs := appendUniqueDB(slices.Clone(projDBs), scopeFromDBs[scopeID]...)
		return projDBs, allDBs
	}
	recordCTE := func(scopeID int, name string, dbs []string) {
		if name == "" || len(dbs) == 0 {
			return
		}
		m := cteByScope[scopeID]
		if m == nil {
			m = make(map[string][]string)
			cteByScope[scopeID] = m
		}
		lower := strings.ToLower(name)
		m[lower] = appendUniqueDB(m[lower], dbs...)
	}
	lookupCTE := func(scopeID int, name string) ([]string, bool) {
		lower := strings.ToLower(name)
		for sid, ok := scopeID, true; ok; sid, ok = parentAliasScope[sid] {
			if m := cteByScope[sid]; m != nil {
				if dbs, hit := m[lower]; hit {
					return dbs, true
				}
			}
		}
		return nil, false
	}
	recordAlias := func(scopeID int, name string, overwrite bool, dbs ...string) {
		if name == "" || len(dbs) == 0 {
			return
		}
		am := aliasesByScope[scopeID]
		if am == nil {
			am = &mysqlAliasMap{
				exact: make(map[string][]string),
				lower: make(map[string][]string),
			}
			aliasesByScope[scopeID] = am
		}
		lower := strings.ToLower(name)
		if overwrite {
			am.exact[name] = appendUniqueDB(nil, dbs...)
			am.lower[lower] = appendUniqueDB(nil, dbs...)
			return
		}
		if _, ok := am.exact[name]; !ok {
			am.exact[name] = appendUniqueDB(nil, dbs...)
		}
		if _, ok := am.lower[lower]; !ok {
			am.lower[lower] = appendUniqueDB(nil, dbs...)
		}
	}

	scopes := []mysqlQualScope{{aliasScopeID: 0, stmtStart: true, allowFromAsTable: true}}
	cur := func() *mysqlQualScope { return &scopes[len(scopes)-1] }
	caretAliasScopeID := 0

	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		sc := cur()
		prevPunct := byte(0)
		if i > 0 {
			prevPunct = toks[i-1].punct
		}

		switch tok.punct {
		case ';':
			aliasID := nextAliasScopeID
			nextAliasScopeID++
			scopes = []mysqlQualScope{{aliasScopeID: aliasID, stmtStart: true, allowFromAsTable: true}}
			if tok.end <= pos {
				caretAliasScopeID = aliasID
			}
			continue
		case '(':
			child := mysqlQualScope{
				aliasScopeID:    sc.aliasScopeID,
				stmtStart:       true,
				inSelectList:    sc.inSelectList,
				pastFromOrWhere: sc.pastFromOrWhere,
			}
			if sc.stmtVerb == "WITH" {
				child.cteName = cteNameBeforeAS(toks, i)
			}
			// Parenthesized table-reference / join group, e.g. FROM (db1.t1 JOIN ...),
			// JOIN (db1.t1), UPDATE [LOW_PRIORITY] [IGNORE] (db1.t1 JOIN ...),
			// DELETE ... USING (db1.t1 JOIN ...), or nested ((db1.t1)) / , (db1.t1).
			if sc.inTableClause && isParenthesizedTablePrefix(toks, i, sc) {
				child.inTableClause = true
				child.allowCommaTable = true
				child.allowFromAsTable = true
				child.inSelectList = false
			} else {
				aliasID := nextAliasScopeID
				nextAliasScopeID++
				parentAliasScope[aliasID] = sc.aliasScopeID
				child.aliasScopeID = aliasID
			}
			scopes = append(scopes, child)
			if tok.end <= pos {
				caretAliasScopeID = child.aliasScopeID
			}
			continue
		case ')':
			var closed mysqlQualScope
			if len(scopes) > 1 {
				closed = scopes[len(scopes)-1]
				scopes = scopes[:len(scopes)-1]
			}
			parent := cur()
			if closed.aliasScopeID != parent.aliasScopeID && !closed.isSubquery && len(scopeFromDBs[closed.aliasScopeID]) == 0 {
				// Expression parentheses inside SELECT (e.g. COALESCE(y.col, 0))
				// defer alias resolution to the enclosing query block's FROM/JOINs.
				if parent.inSelectList {
					for _, ref := range scopeProjRefs[closed.aliasScopeID] {
						recordProjRef(parent.aliasScopeID, ref)
					}
				}
			} else if subProjDBs, subDBs := finalizeScopeDBs(closed.aliasScopeID); len(subDBs) > 0 {
				if closed.aliasScopeID != parent.aliasScopeID {
					if parent.inSelectList {
						for _, db := range subDBs {
							recordProjRef(parent.aliasScopeID, mysqlProjRef{db: db})
						}
						recordScopeFromDB(parent.aliasScopeID, scopeFromDBs[closed.aliasScopeID]...)
					} else if !parent.inTableClause && !parent.pastFromOrWhere && len(scopeProjRefs[parent.aliasScopeID]) == 0 && len(scopeFromDBs[parent.aliasScopeID]) == 0 {
						for _, db := range subProjDBs {
							recordProjRef(parent.aliasScopeID, mysqlProjRef{db: db})
						}
						recordScopeFromDB(parent.aliasScopeID, scopeFromDBs[closed.aliasScopeID]...)
						parent.isSubquery = parent.isSubquery || closed.isSubquery
					} else {
						recordScopeFromDB(parent.aliasScopeID, subDBs...)
					}
				}
				if closed.cteName != "" {
					recordCTE(parent.aliasScopeID, closed.cteName, subDBs)
					recordAlias(parent.aliasScopeID, closed.cteName, true, subDBs...)
				}
				if parent.inTableClause {
					if aliasTok, ok := scanTableAliasAfter(toks, i+1); ok {
						recordAlias(parent.aliasScopeID, aliasTok, true, subDBs...)
					}
				}
			}
			parent.prevKeyword = ""
			parent.stmtStart = false
			if tok.end <= pos {
				caretAliasScopeID = parent.aliasScopeID
			}
			continue
		case ',':
			sc.inTableClause = sc.allowCommaTable
			sc.inInsertReplaceTarget = false
			sc.prevKeyword = ""
			sc.stmtStart = false
			if tok.end <= pos {
				caretAliasScopeID = sc.aliasScopeID
			}
			continue
		case '.':
			sc.prevKeyword = ""
			sc.stmtStart = false
			if tok.end <= pos {
				caretAliasScopeID = sc.aliasScopeID
			}
			continue
		default:
			// Neutral separator or zero punct falls through to word handling.
		}

		if tok.end <= pos {
			caretAliasScopeID = sc.aliasScopeID
		}

		if !tok.isWord {
			continue
		}

		// Record database qualifiers only at the head of a dotted identifier
		// chain (not when preceded by '.'). In DELETE target lists before FROM,
		// ignore `alias.*` wildcards while keeping three-part `db.tbl.*` and `db.`.
		precededByDot := i > 0 && toks[i-1].punct == '.'
		if !precededByDot && i+1 < len(toks) && toks[i+1].punct == '.' {
			isThreePart := i+3 < len(toks) && toks[i+2].isWord && !isClauseBoundaryAfterDot(toks[i+2].upper) && toks[i+3].punct == '.'
			isDeleteStar := sc.stmtVerb == "DELETE" && i+2 < len(toks) && toks[i+2].punct == '*'
			if isThreePart || (sc.inTableClause && !isDeleteStar) {
				addDB(tok.text)
				if sc.inTableClause {
					recordScopeFromDB(sc.aliasScopeID, tok.text)
				} else if sc.inSelectList {
					recordProjRef(sc.aliasScopeID, mysqlProjRef{db: tok.text})
				} else {
					recordScopeFromDB(sc.aliasScopeID, tok.text)
				}
			} else if sc.inSelectList && !isDeleteStar {
				recordProjRef(sc.aliasScopeID, mysqlProjRef{alias: tok.text})
			}
			if sc.inTableClause && !isDeleteStar && i+2 < len(toks) && toks[i+2].isWord && !isClauseBoundaryAfterDot(toks[i+2].upper) {
				recordAlias(sc.aliasScopeID, toks[i+2].text, true, tok.text)
				if aliasTok, ok := scanTableAliasAfter(toks, i+3); ok {
					recordAlias(sc.aliasScopeID, aliasTok, true, tok.text)
				}
			}
		} else if sc.inTableClause && !precededByDot && (i+1 >= len(toks) || toks[i+1].punct != '.') &&
			isTableStartPosition(sc.prevKeyword, prevPunct) && !isNonAliasClauseKeyword(tok.upper) {
			targetDBs, fromCTE := lookupCTE(sc.aliasScopeID, tok.text)
			if !fromCTE && defaultDB != "" {
				targetDBs = []string{defaultDB}
			}
			if len(targetDBs) > 0 {
				recordScopeFromDB(sc.aliasScopeID, targetDBs...)
				recordAlias(sc.aliasScopeID, tok.text, fromCTE, targetDBs...)
				if aliasTok, ok := scanTableAliasAfter(toks, i+1); ok {
					recordAlias(sc.aliasScopeID, aliasTok, fromCTE, targetDBs...)
				}
			}
		}

		// Keyword-driven clause tracking. Backtick-quoted words have tok.upper == "".
		// An unquoted member after '.' (e.g. SELECT u. FROM otherdb.users u) only
		// triggers clause transitions when it is a reserved clause boundary.
		if tok.upper == "" || (precededByDot && !isClauseBoundaryAfterDot(tok.upper)) {
			sc.prevKeyword = ""
			sc.stmtStart = false
			continue
		}

		kw := tok.upper
		wasStmtStart := sc.stmtStart
		if wasStmtStart {
			sc.stmtVerb = kw
		} else if sc.stmtVerb == "EXPLAIN" {
			switch kw {
			case "SELECT", "UPDATE", "DELETE", "INSERT", "REPLACE", "TABLE":
				sc.stmtVerb = kw
			default:
			}
		}
		prevKW := sc.prevKeyword
		sc.stmtStart = false
		sc.prevKeyword = kw

		// While scanning privilege names/clauses before ON in GRANT/REVOKE, keep
		// privilege keywords (e.g. SELECT, INSERT, UPDATE, TRIGGER, REFERENCES)
		// from opening a table clause until ON arrives.
		if sc.inGrantPrivList {
			if kw == "ON" {
				sc.inGrantPrivList = false
				sc.inTableClause = true
				sc.allowCommaTable = false
				sc.afterDDLOnObj = false
			}
			continue
		}

		switch kw {
		case "SELECT", "WITH":
			if len(scopes) > 1 {
				// Derived tables / subqueries isolate their table aliases from outer scopes.
				if _, ok := parentAliasScope[sc.aliasScopeID]; !ok && sc.aliasScopeID == scopes[len(scopes)-2].aliasScopeID {
					aliasID := nextAliasScopeID
					nextAliasScopeID++
					parentAliasScope[aliasID] = sc.aliasScopeID
					sc.aliasScopeID = aliasID
					if tok.end <= pos {
						caretAliasScopeID = aliasID
					}
				}
			}
			sc.inSelectList = kw == "SELECT"
			sc.pastFromOrWhere = false
			if kw == "SELECT" {
				sc.isSubquery = true
			}
			sc.inTableClause = false
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = false
			sc.allowFromAsTable = true
			sc.afterCreate = false
			sc.afterRename = false
			sc.afterDDLOnObj = false
		case "WHERE", "SET", "VALUES", "VALUE", "GROUP", "HAVING", "ORDER", "LIMIT",
			"UNION", "INTERSECT", "EXCEPT", "FOR", "WINDOW", "RETURNING", "RETURN":
			if sc.isSubquery || len(scopes) == 1 {
				sc.inSelectList = false
				sc.pastFromOrWhere = true
			}
			sc.inTableClause = false
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = false
		case "ON":
			sc.inTableClause = sc.afterDDLOnObj
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = false
			sc.afterDDLOnObj = false
		case "FROM":
			if sc.isSubquery || len(scopes) == 1 {
				sc.inSelectList = false
				sc.pastFromOrWhere = true
			}
			if sc.stmtVerb == "REVOKE" {
				sc.inTableClause = false
				sc.allowCommaTable = false
			} else if sc.allowFromAsTable || sc.stmtVerb == "SHOW" || sc.stmtVerb == "DELETE" || prevKW == "COLUMNS" || prevKW == "FIELDS" || prevKW == "INDEX" || prevKW == "INDEXES" || prevKW == "KEYS" {
				sc.inTableClause = true
				sc.allowCommaTable = true
				sc.inInsertReplaceTarget = false
			}
		case "JOIN", "STRAIGHT_JOIN":
			sc.inTableClause = true
			sc.allowCommaTable = true
			sc.inInsertReplaceTarget = false
		case "UPDATE":
			if prevKW != "ON" && prevKW != "KEY" && prevKW != "FOR" && prevKW != "BEFORE" && prevKW != "AFTER" {
				sc.stmtVerb = "UPDATE"
				sc.inTableClause = true
				sc.allowCommaTable = true
				sc.inInsertReplaceTarget = false
			}
		case "DELETE":
			if prevKW != "ON" && prevKW != "BEFORE" && prevKW != "AFTER" {
				sc.stmtVerb = "DELETE"
				sc.inTableClause = true
				sc.allowCommaTable = true
				sc.inInsertReplaceTarget = false
			}
		case "INSERT", "REPLACE":
			sc.inTableClause = true
			sc.inInsertReplaceTarget = true
			sc.allowCommaTable = false
		case "INTO":
			if sc.inInsertReplaceTarget || sc.stmtVerb == "IMPORT" {
				sc.inTableClause = true
				sc.allowCommaTable = false
				sc.inInsertReplaceTarget = false
			} else {
				if sc.isSubquery || len(scopes) == 1 {
					sc.inSelectList = false
					sc.pastFromOrWhere = true
				}
				sc.inTableClause = false
				sc.allowCommaTable = false
			}
		case "USING":
			sc.inInsertReplaceTarget = false
			if sc.stmtVerb != "DELETE" && i+1 < len(toks) && toks[i+1].punct == '(' {
				sc.inTableClause = false
				sc.allowCommaTable = false
			} else {
				sc.inTableClause = true
				sc.allowCommaTable = true
			}
		case "CREATE":
			sc.afterCreate = true
			sc.inTableClause = false
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = false
		case "RENAME":
			sc.afterRename = true
			sc.inTableClause = false
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = false
		case "GRANT", "REVOKE":
			sc.inGrantPrivList = true
			sc.inTableClause = false
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = false
			sc.allowFromAsTable = false
		case "INDEX", "TRIGGER":
			sc.inInsertReplaceTarget = false
			if kw == "INDEX" && sc.stmtVerb == "CACHE" && prevKW == "CACHE" {
				sc.inTableClause = true
				sc.allowCommaTable = true
				sc.afterDDLOnObj = false
			} else {
				sc.afterDDLOnObj = true
				if kw == "TRIGGER" {
					sc.inTableClause = true
					sc.allowCommaTable = false
				}
			}
		case "CACHE":
			if sc.stmtVerb == "LOAD" && prevKW == "INTO" {
				sc.inTableClause = true
				sc.allowCommaTable = true
				sc.inInsertReplaceTarget = false
			}
		case "TABLE", "TABLES":
			sc.inTableClause = true
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = isMultiTableStatementVerb(sc.stmtVerb)
		case "VIEW":
			sc.inTableClause = true
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = sc.stmtVerb == "DROP"
		case "SEQUENCE":
			sc.inTableClause = true
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = sc.stmtVerb == "DROP"
		case "TRUNCATE":
			if (wasStmtStart || prevKW == "BEGIN") && (i+1 >= len(toks) || toks[i+1].punct != '(') {
				sc.inTableClause = true
				sc.inInsertReplaceTarget = false
				sc.allowCommaTable = false
			}
		case "EVENT", "FUNCTION", "PROCEDURE", "CALL", "HANDLER", "REFERENCES":
			sc.inTableClause = true
			sc.inInsertReplaceTarget = false
			sc.allowCommaTable = false
		case "DESC", "DESCRIBE", "EXPLAIN":
			if wasStmtStart {
				sc.inTableClause = true
				sc.allowCommaTable = false
			}
		case "TO":
			if sc.afterRename {
				sc.inTableClause = true
			} else {
				sc.inTableClause = false
				sc.allowCommaTable = false
			}
		case "LIKE":
			if sc.afterCreate {
				sc.inTableClause = true
			}
		case "IN":
			if sc.stmtVerb == "CACHE" {
				sc.inTableClause = false
				sc.allowCommaTable = false
			} else if prevKW == "COLUMNS" || prevKW == "FIELDS" || prevKW == "INDEX" || prevKW == "INDEXES" || prevKW == "KEYS" {
				sc.inTableClause = true
				sc.allowCommaTable = false
			}
		default:
			// Non-clause keywords leave table-clause state unchanged.
		}
	}

	var caretDBs []string
	if caretIdent := databaseQualifierAtCaret(statement, pos); caretIdent != "" {
		if caretIdent == defaultDB {
			caretDBs = []string{defaultDB}
		} else {
			for _, q := range out {
				if q == caretIdent {
					caretDBs = []string{q}
					break
				}
			}
		}
		if len(caretDBs) == 0 {
			if dbs, hit := resolveAliasDBs(caretAliasScopeID, caretIdent); hit {
				caretDBs = dbs
			}
		}
	}
	return out, caretDBs
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
			if q[0] != '`' && at > 0 {
				prev, _ := utf8.DecodeLastRuneInString(s[:at])
				if isIdentRune(prev) {
					continue
				}
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

func isMultiTableStatementVerb(verb string) bool {
	switch verb {
	case "DROP", "RENAME", "LOCK", "TRUNCATE", "ANALYZE", "CHECK", "CHECKSUM", "OPTIMIZE", "REPAIR", "FLUSH":
		return true
	default:
		return false
	}
}

// databaseQualifierAtCaret returns the leading database identifier of a dotted
// qualifier chain active at pos (e.g. `Foo` for `Foo.t.` or `Foo.`), or "".
func databaseQualifierAtCaret(statement string, pos int) string {
	pos = max(0, min(pos, len(statement)))
	prefix := stripStringsAndComments(statement[:pos])
	toks := scanMySQLQualTokens(prefix)
	if len(toks) < 2 {
		return ""
	}
	trimmed := strings.TrimRight(prefix, " \t\r\n")
	if trimmed == "" {
		return ""
	}
	lastIdx := len(toks) - 1
	if trimmed[len(trimmed)-1] == '.' {
		if toks[lastIdx].punct != '.' {
			return ""
		}
		lastIdx--
	} else if isSpaceByte(prefix[len(prefix)-1]) || !toks[lastIdx].isWord || toks[lastIdx-1].punct != '.' {
		return ""
	}
	if lastIdx < 0 || !toks[lastIdx].isWord {
		return ""
	}
	for steps := 0; steps < 2 && lastIdx >= 2 && toks[lastIdx-1].punct == '.' && toks[lastIdx-2].isWord; steps++ {
		lastIdx -= 2
	}
	return toks[lastIdx].text
}

func isParenthesizedTablePrefix(toks []mysqlQualToken, parenIdx int, sc *mysqlQualScope) bool {
	if parenIdx == 0 {
		return true
	}
	prev := toks[parenIdx-1]
	if prev.punct == '(' || prev.punct == ',' {
		return true
	}
	switch prev.upper {
	case "FROM", "JOIN", "STRAIGHT_JOIN", "UPDATE":
		return true
	case "USING":
		return sc.stmtVerb == "DELETE"
	case "LOW_PRIORITY", "IGNORE":
		return sc.stmtVerb == "UPDATE"
	default:
		return false
	}
}

func isTableStartPosition(prevKW string, prevPunct byte) bool {
	if prevPunct == ',' {
		return true
	}
	switch prevKW {
	case "FROM", "JOIN", "STRAIGHT_JOIN", "UPDATE", "INTO":
		return true
	default:
		return false
	}
}

func scanTableAliasAfter(toks []mysqlQualToken, startIdx int) (string, bool) {
	if startIdx >= len(toks) {
		return "", false
	}
	idx := startIdx
	if toks[idx].upper == "AS" {
		idx++
	}
	if idx >= len(toks) || !toks[idx].isWord || isNonAliasClauseKeyword(toks[idx].upper) {
		return "", false
	}
	return toks[idx].text, true
}

func cteNameBeforeAS(toks []mysqlQualToken, openParenIdx int) string {
	if openParenIdx < 2 || toks[openParenIdx-1].upper != "AS" {
		return ""
	}
	k := openParenIdx - 2
	if toks[k].punct == ')' {
		depth := 1
		k--
		for k >= 0 && depth > 0 {
			switch toks[k].punct {
			case ')':
				depth++
			case '(':
				depth--
			default:
			}
			k--
		}
	}
	if k >= 0 && toks[k].isWord && toks[k].upper != "WITH" && toks[k].upper != "RECURSIVE" {
		return toks[k].text
	}
	return ""
}

func isNonAliasClauseKeyword(upper string) bool {
	switch upper {
	case "AS", "ON", "USING", "WHERE", "SET", "VALUES", "VALUE",
		"GROUP", "HAVING", "ORDER", "LIMIT", "OFFSET", "FOR", "LOCK",
		"UNION", "INTERSECT", "EXCEPT", "WINDOW", "PARTITION",
		"USE", "IGNORE", "FORCE", "JOIN", "INNER", "LEFT", "RIGHT",
		"FULL", "CROSS", "NATURAL", "STRAIGHT_JOIN", "SELECT", "FROM",
		"INTO", "TO", "RETURNING", "RETURN", "LOW_PRIORITY", "DELAYED",
		"HIGH_PRIORITY", "QUICK", "ALL", "DISTINCT", "WITH":
		return true
	default:
		return false
	}
}

func isClauseBoundaryAfterDot(upper string) bool {
	switch upper {
	case "SELECT", "WITH", "FROM", "JOIN", "STRAIGHT_JOIN",
		"LEFT", "RIGHT", "INNER", "CROSS", "NATURAL",
		"WHERE", "SET", "VALUES", "GROUP", "HAVING", "ORDER", "LIMIT",
		"UNION", "INTERSECT", "EXCEPT", "ON", "USING", "INTO", "FOR",
		"WINDOW", "RETURNING":
		return true
	default:
		return false
	}
}

func isIdentByte(b byte) bool {
	return b == '_' || b == '$' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$'
}

func nextIdentRune(s string, i int) (int, bool) {
	if s[i] < utf8.RuneSelf {
		return 1, isIdentByte(s[i])
	}
	r, size := utf8.DecodeRuneInString(s[i:])
	return size, r != utf8.RuneError && isIdentRune(r)
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
		if !isIdentRune(r) {
			return true
		}
	}
	firstRune, _ := utf8.DecodeRuneInString(name)
	if unicode.IsDigit(firstRune) {
		return true
	}
	return isReservedKeyword != nil && isReservedKeyword(name)
}

func caretAtStatementStart(statement string, pos int) bool {
	pos = max(0, min(pos, len(statement)))
	before := stripStringsAndComments(statement)[:pos]
	before = strings.TrimRightFunc(before, isIdentRune)
	before = strings.TrimRight(before, " \t\r\n")
	if before == "" || before[len(before)-1] == ';' {
		return true
	}
	return caretAfterWithClause(before)
}

func caretAfterWithClause(before string) bool {
	toks := scanMySQLQualTokens(before)
	if len(toks) == 0 || toks[len(toks)-1].punct != ')' {
		return false
	}
	depth := 0
	startIdx := 0
	withIdx := -1
	withDepth := -1
	for i, tok := range toks {
		switch tok.punct {
		case ';':
			startIdx = i + 1
			depth = 0
			withIdx = -1
			withDepth = -1
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
			if depth < withDepth {
				withIdx = -1
				withDepth = -1
			}
		default:
			if tok.upper == "WITH" && (i == startIdx || (i > 0 && toks[i-1].punct == '(')) {
				withIdx = i
				withDepth = depth
			}
		}
	}
	if withIdx < 0 || depth != withDepth {
		return false
	}
	d := withDepth
	sawCTEBody := false
	for i := withIdx + 1; i < len(toks); i++ {
		tok := toks[i]
		switch tok.punct {
		case '(':
			if d == withDepth && i > 0 && toks[i-1].upper == "AS" {
				sawCTEBody = true
			}
			d++
		case ')':
			if d > withDepth {
				d--
			}
		case ',':
			if d == withDepth {
				sawCTEBody = false
			}
		default:
			if d == withDepth && sawCTEBody && tok.isWord {
				return false
			}
		}
	}
	return sawCTEBody
}

// CaretInsideMySQLBacktickIdentifier reports whether the caret sits inside an
// open backtick-quoted identifier (an odd number of backticks precede it).
func CaretInsideMySQLBacktickIdentifier(statement string, pos int) bool {
	pos = max(0, min(pos, len(statement)))
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

// lineOffsetToBytePos converts a 1-based line and 0-based UTF-16 code-unit
// column offset into a byte offset into s, matching LSP/Monaco positions.
func lineOffsetToBytePos(s string, line, offset int) int {
	currentLine := 1
	col := 0
	for i := 0; i < len(s); {
		if currentLine == line && col >= offset {
			return i
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\n' {
			if currentLine == line {
				return i
			}
			currentLine++
			col = 0
		} else if r <= 0xFFFF {
			col++
		} else {
			col += 2
		}
		i += size
	}
	return len(s)
}
