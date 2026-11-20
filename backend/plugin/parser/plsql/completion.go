package plsql

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
	oraclecompletion "github.com/bytebase/omni/oracle/completion"
	oracleparser "github.com/bytebase/omni/oracle/parser"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(store.Engine_ORACLE, Completion)
}

var oracleQuerySceneStartKeywords = map[string]bool{
	"EXPLAIN": true,
	"SELECT":  true,
	"WITH":    true,
}

// Completion returns Oracle auto-completion candidates backed by omni's
// parser-native Oracle completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	sql, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, false /* tricky */)

	var tokens []oracleparser.Token
	if cCtx.Scene == base.SceneTypeQuery {
		tokens = oracleparser.Tokenize(sql)
		if querySceneDisallowsCompletion(tokens, byteOffset) {
			return nil, nil
		}
	}

	opts := buildCompletionOptions(ctx, cCtx)
	raw := oraclecompletion.Complete(sql, byteOffset, nil, opts...)
	caretInQuotes := caretInsideDoubleQuotes(sql, byteOffset)

	seen := make(map[string]base.Candidate, len(raw))
	for _, c := range raw {
		typ := convertCandidateType(c.Type)
		if typ == base.CandidateTypeNone {
			continue
		}
		candidate := base.Candidate{
			Type:       typ,
			Text:       formatCandidateText(c, caretInQuotes),
			Definition: c.Definition,
			Comment:    c.Comment,
		}
		key := candidate.String()
		if _, exists := seen[key]; !exists {
			seen[key] = candidate
		}
	}

	result := make([]base.Candidate, 0, len(seen))
	for _, candidate := range seen {
		result = append(result, candidate)
	}
	slices.SortFunc(result, compareCandidates)

	if cCtx.Scene == base.SceneTypeQuery {
		result = filterCandidatesByScene(sql, tokens, byteOffset, result)
	}
	return result, nil
}

func buildCompletionOptions(ctx context.Context, cCtx base.CompletionContext) []oraclecompletion.Option {
	var opts []oraclecompletion.Option
	if cCtx.DefaultDatabase != "" {
		opts = append(opts, oraclecompletion.WithDefaultSchema(cCtx.DefaultDatabase))
	}
	if cCtx.ListDatabaseNames != nil {
		if names, err := cCtx.ListDatabaseNames(ctx, cCtx.InstanceID); err == nil && len(names) > 0 {
			opts = append(opts, oraclecompletion.WithSchemaNames(names))
		}
	}
	if cCtx.Metadata != nil {
		cache := make(map[string]*metadatapb.DatabaseSchemaMetadata)
		loadSchema := func(schema string) *metadatapb.DatabaseSchemaMetadata {
			if schema == "" {
				return nil
			}
			if cached, ok := cache[schema]; ok {
				return cached
			}
			_, md, err := cCtx.Metadata(ctx, cCtx.InstanceID, schema)
			if err != nil || md == nil {
				cache[schema] = nil
				return nil
			}
			protoMeta := md.GetProto()
			cache[schema] = protoMeta
			return protoMeta
		}
		if cCtx.DefaultDatabase != "" {
			if defaultMeta := loadSchema(cCtx.DefaultDatabase); defaultMeta != nil {
				opts = append(opts, oraclecompletion.WithSchema(cCtx.DefaultDatabase, defaultMeta))
			}
		}
		opts = append(opts, oraclecompletion.WithSchemaLoader(loadSchema))
	}
	return opts
}

func convertCandidateType(t oraclecompletion.CandidateType) base.CandidateType {
	switch t {
	case oraclecompletion.CandidateKeyword, oraclecompletion.CandidateType_:
		return base.CandidateTypeKeyword
	case oraclecompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case oraclecompletion.CandidateTable, oraclecompletion.CandidateSynonym:
		return base.CandidateTypeTable
	case oraclecompletion.CandidateView:
		return base.CandidateTypeView
	case oraclecompletion.CandidateMaterializedView:
		return base.CandidateTypeMaterializedView
	case oraclecompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case oraclecompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case oraclecompletion.CandidateProcedure:
		return base.CandidateTypeRoutine
	case oraclecompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case oraclecompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case oraclecompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	case oraclecompletion.CandidateVariable:
		return base.CandidateTypeUserVar
	default:
		return base.CandidateTypeNone
	}
}

func formatCandidateText(c oraclecompletion.Candidate, caretInQuotes bool) string {
	switch c.Type {
	case oraclecompletion.CandidateKeyword, oraclecompletion.CandidateType_:
		return c.Text
	case oraclecompletion.CandidateFunction:
		if strings.HasSuffix(c.Text, "()") {
			return c.Text
		}
		return c.Text + "()"
	default:
		if caretInQuotes || !identifierNeedsQuoting(c.Text) {
			return c.Text
		}
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(c.Text, `"`, `""`))
	}
}

func identifierNeedsQuoting(s string) bool {
	if s == "" {
		return true
	}
	if oracleparser.IsReservedKeyword(s) {
		return true
	}
	if s != strings.ToUpper(s) {
		return true
	}
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) {
			return true
		}
		if i > 0 && !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' && r != '#' {
			return true
		}
	}
	return false
}

func caretInsideDoubleQuotes(sql string, byteOffset int) bool {
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}
	i := 0
	for i < byteOffset {
		switch {
		case isQQuoteStart(sql, i, byteOffset):
			prefixLen := 2
			if sql[i] == 'n' || sql[i] == 'N' {
				prefixLen = 3
			}
			delimPos := i + prefixLen
			if delimPos >= byteOffset {
				return false
			}
			openDelim := sql[delimPos]
			if openDelim == ' ' || openDelim == '\t' || openDelim == '\n' || openDelim == '\r' || openDelim == '\f' {
				i = byteOffset
				continue
			}
			closeDelim := qQuoteCloseDelim(openDelim)
			j := delimPos + 1
			closed := false
			for j+1 < byteOffset {
				if sql[j] == closeDelim && sql[j+1] == '\'' {
					i = j + 2
					closed = true
					break
				}
				j++
			}
			if !closed {
				return false
			}
		case sql[i] == '\'':
			i++
			for i < byteOffset {
				if sql[i] == '\'' {
					i++
					if i < byteOffset && sql[i] == '\'' {
						i++
						continue
					}
					break
				}
				i++
			}
		case sql[i] == '"':
			i++
			closed := false
			for i < byteOffset {
				if sql[i] == '"' {
					i++
					if i < byteOffset && sql[i] == '"' {
						i++
						continue
					}
					closed = true
					break
				}
				i++
			}
			if !closed {
				return true
			}
		case sql[i] == '-' && i+1 < byteOffset && sql[i+1] == '-':
			i += 2
			for i < byteOffset && sql[i] != '\n' && sql[i] != '\r' {
				i++
			}
		case sql[i] == '/' && i+1 < byteOffset && sql[i+1] == '*':
			i += 2
			for i+1 < byteOffset {
				if sql[i] == '*' && sql[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
		default:
			i++
		}
	}
	return false
}

func isQQuoteStart(sql string, i, limit int) bool {
	if i > 0 && isOracleIdentByte(sql[i-1]) {
		return false
	}
	if (sql[i] == 'q' || sql[i] == 'Q') && i+1 < limit && sql[i+1] == '\'' {
		return true
	}
	if (sql[i] == 'n' || sql[i] == 'N') && i+2 < limit && (sql[i+1] == 'q' || sql[i+1] == 'Q') && sql[i+2] == '\'' {
		return true
	}
	return false
}

func qQuoteCloseDelim(openDelim byte) byte {
	switch openDelim {
	case '[':
		return ']'
	case '{':
		return '}'
	case '(':
		return ')'
	case '<':
		return '>'
	default:
		return openDelim
	}
}

func compareCandidates(i, j base.Candidate) int {
	if i.Type != j.Type {
		if i.Type < j.Type {
			return -1
		}
		return 1
	}
	if i.Text != j.Text {
		if i.Text < j.Text {
			return -1
		}
		return 1
	}
	if i.Definition < j.Definition {
		return -1
	}
	if i.Definition > j.Definition {
		return 1
	}
	return 0
}

func filterCandidatesByScene(sql string, tokens []oracleparser.Token, byteOffset int, candidates []base.Candidate) []base.Candidate {
	if !isAtStatementStartCompletion(sql, tokens, byteOffset) {
		return candidates
	}
	result := make([]base.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Type == base.CandidateTypeKeyword && !oracleQuerySceneStartKeywords[strings.ToUpper(candidate.Text)] {
			continue
		}
		result = append(result, candidate)
	}
	return result
}

func querySceneDisallowsCompletion(tokens []oracleparser.Token, byteOffset int) bool {
	idx := currentStatementMainTokenIndex(tokens, byteOffset)
	if idx < 0 || tokens[idx].End >= byteOffset {
		return false
	}
	if strings.EqualFold(tokens[idx].Str, "EXPLAIN") {
		return false
	}
	switch tokens[idx].Type {
	case oracleparser.SELECT, oracleparser.WITH:
		return false
	default:
		return true
	}
}

func isAtStatementStartCompletion(sql string, tokens []oracleparser.Token, byteOffset int) bool {
	collectOffset := byteOffset
	for collectOffset > 0 && isOracleIdentByte(sql[collectOffset-1]) {
		collectOffset--
	}
	start := currentStatementStartTokenIndex(tokens, collectOffset)
	if start >= len(tokens) {
		return true
	}
	token := tokens[start]
	return token.Type == ';' || token.Loc >= collectOffset
}

func isOracleIdentByte(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '$' || c == '#' || c >= 128
}

func currentStatementStartTokenIndex(tokens []oracleparser.Token, offset int) int {
	start := 0
	for i, token := range tokens {
		if token.Loc >= offset {
			break
		}
		if token.Type == ';' {
			start = i + 1
		}
	}
	return start
}

func currentStatementMainTokenIndex(tokens []oracleparser.Token, byteOffset int) int {
	start := currentStatementStartTokenIndex(tokens, byteOffset)
	if start >= len(tokens) {
		return -1
	}
	token := tokens[start]
	if token.Loc >= byteOffset || token.Type == ';' {
		return -1
	}
	if token.Type == oracleparser.WITH {
		depth := 0
		for i := start + 1; i < len(tokens); i++ {
			t := tokens[i]
			if t.Loc >= byteOffset || t.Type == ';' {
				break
			}
			switch t.Type {
			case '(':
				depth++
			case ')':
				if depth > 0 {
					depth--
				}
			default:
				if depth == 0 && (t.Type == oracleparser.SELECT || t.Type == oracleparser.INSERT ||
					t.Type == oracleparser.UPDATE || t.Type == oracleparser.DELETE || t.Type == oracleparser.MERGE) {
					return i
				}
			}
		}
	}
	return start
}

func computeSQLAndByteOffset(statement string, caretLine int, caretOffset int, _ bool) (string, int) {
	sql, newLine, newOffset := skipHeadingSQLs(statement, caretLine, caretOffset)
	return sql, lineColumnToByteOffset(sql, newLine, newOffset)
}

func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	newCaretLine, newCaretOffset := caretLine, caretOffset
	list, err := SplitSQLForCompletion(statement)
	if err != nil || notEmptySQLCount(list) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLine--
	start := 0
	for i, sql := range list {
		sqlEndLine := int(sql.End.GetLine()) - 1
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLine || (sqlEndLine == caretLine && sqlEndColumn >= caretOffset) {
			start = i
			if i == 0 {
				break
			}
			previousSQLEndLine := int(list[i-1].End.GetLine()) - 1
			previousSQLEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLine - previousSQLEndLine + 1
			if caretLine == previousSQLEndLine {
				newCaretOffset = caretOffset - previousSQLEndColumn + 1
			}
			break
		}
	}

	var buf strings.Builder
	for i := start; i < len(list); i++ {
		if _, err := buf.WriteString(list[i].Text); err != nil {
			return statement, caretLine + 1, caretOffset
		}
	}
	return buf.String(), newCaretLine, newCaretOffset
}

func notEmptySQLCount(list []base.Statement) int {
	count := 0
	for _, sql := range list {
		if !sql.Empty {
			count++
		}
	}
	return count
}

func lineColumnToByteOffset(sql string, line, column int) int {
	currentLine := 1
	for i := 0; i < len(sql); i++ {
		if currentLine == line {
			pos := i
			for c := 0; c < column && pos < len(sql); {
				r, size := utf8.DecodeRuneInString(sql[pos:])
				if r == '\n' {
					return pos
				}
				unitCount := 1
				if r > 0xFFFF {
					unitCount = 2
				}
				if c+unitCount > column {
					return pos
				}
				pos += size
				c += unitCount
			}
			if pos > len(sql) {
				return len(sql)
			}
			return pos
		}
		if sql[i] == '\n' {
			currentLine++
		}
	}
	return len(sql)
}
