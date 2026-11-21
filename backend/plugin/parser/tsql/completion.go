package tsql

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	mssqlcatalog "github.com/bytebase/omni/mssql/catalog"
	mssqlcompletion "github.com/bytebase/omni/mssql/completion"
	mssqlparser "github.com/bytebase/omni/mssql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_MSSQL, Completion)
}

// Completion returns T-SQL auto-completion candidates backed by omni's
// parser-native MSSQL completer and catalog.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	sql, line, offset := skipHeadingSQLs(statement, caretLine, caretOffset)
	byteOffset := lineColumnToByteOffset(sql, line, offset)

	cat := buildCompletionCatalog(ctx, cCtx, sql)
	caretQuoted := caretTokenIsQuoted(sql, byteOffset)
	raw := mssqlcompletion.Complete(sql, byteOffset, cat)

	seen := make(map[string]base.Candidate, len(raw))
	for _, c := range raw {
		typ := convertCandidateType(c.Type)
		if typ == base.CandidateTypeNone {
			continue
		}
		text := formatCandidateText(c, typ, caretQuoted)
		candidate := base.Candidate{
			Type:       typ,
			Text:       text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}
		key := candidate.String()
		if _, exists := seen[key]; !exists {
			seen[key] = candidate
		}
	}

	result := make([]base.Candidate, 0, len(seen))
	for _, c := range seen {
		result = append(result, c)
	}
	slices.SortFunc(result, compareCandidates)
	return result, nil
}

func buildCompletionCatalog(ctx context.Context, cCtx base.CompletionContext, statement string) *mssqlcatalog.Catalog {
	if cCtx.Metadata == nil && cCtx.ListDatabaseNames == nil {
		return nil
	}

	var allNames []string
	if cCtx.ListDatabaseNames != nil {
		if names, err := cCtx.ListDatabaseNames(ctx, cCtx.InstanceID); err == nil {
			allNames = names
		}
	}

	defaultDB := cCtx.DefaultDatabase
	for _, name := range allNames {
		if strings.EqualFold(name, defaultDB) {
			defaultDB = name
			break
		}
	}

	cat := mssqlcatalog.New()
	if defaultDB != "" {
		loadDatabaseIntoCatalog(ctx, cCtx, cat, defaultDB, true)
	}
	if cCtx.DefaultSchema != "" {
		cat.SetDefaultSchema(cCtx.DefaultSchema)
	}

	lowerStmt := strings.ToLower(statement)
	for _, name := range allNames {
		if strings.EqualFold(name, defaultDB) {
			continue
		}
		needFullLoad := name != "" && strings.Contains(lowerStmt, strings.ToLower(name))
		loadDatabaseIntoCatalog(ctx, cCtx, cat, name, needFullLoad)
	}
	return cat
}

func loadDatabaseIntoCatalog(ctx context.Context, cCtx base.CompletionContext, cat *mssqlcatalog.Catalog, dbName string, full bool) {
	if full && cCtx.Metadata != nil {
		if _, meta, err := cCtx.Metadata(ctx, cCtx.InstanceID, dbName); err == nil && meta != nil {
			_ = cat.LoadMetadataInto(ctx, meta.GetProto(), dbName)
			return
		}
	}
	_ = cat.LoadMetadataInto(ctx, nil, dbName)
}

func formatCandidateText(c mssqlcompletion.Candidate, typ base.CandidateType, caretQuoted bool) string {
	switch {
	case c.Type == mssqlcompletion.CandidateKeyword || c.Type == mssqlcompletion.CandidateType_:
		return c.Text
	case typ == base.CandidateTypeFunction:
		if strings.HasSuffix(c.Text, "()") {
			return c.Text
		}
		return c.Text + "()"
	case c.Text == "$action":
		return c.Text
	default:
		if caretQuoted {
			return c.Text
		}
		if !isRegularIdentifier(c.Text) {
			return fmt.Sprintf("[%s]", c.Text)
		}
		return c.Text
	}
}

func isRegularIdentifier(identifier string) bool {
	if len(identifier) == 0 {
		return true
	}

	firstChar := rune(identifier[0])
	if !unicode.IsLetter(firstChar) && firstChar != '_' && firstChar != '@' && firstChar != '#' {
		return false
	}
	for _, r := range identifier[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '@' && r != '$' && r != '#' && r != '_' {
			return false
		}
	}
	if IsTSQLReservedKeyword(identifier, false) {
		return false
	}
	for _, r := range identifier {
		if r == ' ' || !unicode.IsPrint(r) {
			return false
		}
	}
	return utf8.ValidString(identifier)
}

func caretTokenIsQuoted(sql string, cursorByteOffset int) bool {
	tokens := mssqlparser.Tokenize(sql)
	caretIdx := len(tokens)
	for i, tok := range tokens {
		if tok.Loc >= cursorByteOffset {
			caretIdx = i
			break
		}
	}
	checkIdx := func(idx int) bool {
		if idx < 0 || idx >= len(tokens) {
			return false
		}
		tok := tokens[idx]
		if !mssqlparser.IsIdentTokenType(tok.Type) || tok.Loc >= len(sql) {
			return false
		}
		return sql[tok.Loc] == '"' || sql[tok.Loc] == '['
	}
	if checkIdx(caretIdx) {
		return true
	}
	if caretIdx > 0 && tokens[caretIdx-1].End >= cursorByteOffset {
		return checkIdx(caretIdx - 1)
	}
	return false
}

func convertCandidateType(t mssqlcompletion.CandidateType) base.CandidateType {
	switch t {
	case mssqlcompletion.CandidateKeyword, mssqlcompletion.CandidateType_:
		return base.CandidateTypeKeyword
	case mssqlcompletion.CandidateDatabase:
		return base.CandidateTypeDatabase
	case mssqlcompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case mssqlcompletion.CandidateTable:
		return base.CandidateTypeTable
	case mssqlcompletion.CandidateView:
		return base.CandidateTypeView
	case mssqlcompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case mssqlcompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case mssqlcompletion.CandidateProcedure:
		return base.CandidateTypeRoutine
	case mssqlcompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case mssqlcompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case mssqlcompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	default:
		return base.CandidateTypeNone
	}
}

func compareCandidates(a, b base.Candidate) int {
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
}

func lineColumnToByteOffset(sql string, line, column int) int {
	currentLine := 1
	for i := 0; i < len(sql); i++ {
		if currentLine == line {
			pos := i
			for c := 0; c < column && pos < len(sql); c++ {
				_, size := utf8.DecodeRuneInString(sql[pos:])
				pos += size
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

func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	list, err := SplitSQL(statement)
	if err != nil {
		return statement, caretLine, caretOffset
	}
	return base.StatementAtCaret(list, statement, caretLine, caretOffset, true)
}
