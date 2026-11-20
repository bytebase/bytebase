package pg

import (
	"context"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	pgast "github.com/bytebase/omni/pg/ast"
	pgcatalog "github.com/bytebase/omni/pg/catalog"
	pgcompletion "github.com/bytebase/omni/pg/completion"
	pgparser "github.com/bytebase/omni/pg/parser"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(store.Engine_POSTGRES, Completion)
	base.RegisterCompleteFunc(store.Engine_SNOWFLAKE, Completion)
	base.RegisterCompleteFunc(store.Engine_COCKROACHDB, Completion)
}

// Completion returns PostgreSQL auto-completion candidates backed by omni's
// parser-native completer. Bytebase metadata is loaded into an omni PostgreSQL
// catalog so the completer can resolve schemas, relations, columns, sequences,
// functions, and PostgreSQL grammar slots.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	cat := buildCompletionCatalog(ctx, cCtx)

	sql, line, offset := completionStatementAtCaret(statement, caretLine, caretOffset)
	byteOffset := caretToByteOffset(sql, line, offset)
	caretInQuotes := caretInsideDoubleQuotes(sql, byteOffset)
	candidates := pgcompletion.Complete(sql, byteOffset, cat)

	result := make([]base.Candidate, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		candidate := base.Candidate{
			Text:       completionCandidateText(c, caretInQuotes),
			Type:       convertCandidateType(c.Type),
			Definition: c.Definition,
			Comment:    c.Comment,
		}
		key := candidate.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	slices.SortFunc(result, func(a, b base.Candidate) int {
		if a.Type != b.Type {
			return strings.Compare(string(a.Type), string(b.Type))
		}
		return strings.Compare(a.Text, b.Text)
	})
	return result, nil
}

// completionStatementAtCaret isolates the statement containing the caret so
// neither table refs from earlier statements nor broken trailing statements in
// the same editor buffer affect completion.
func completionStatementAtCaret(statement string, caretLine int, caretOffset int) (string, int, int) {
	list, err := SplitSQL(statement)
	if err != nil || len(base.FilterEmptyStatements(list)) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLineZeroBased := caretLine - 1
	start := 0
	newCaretLine, newCaretOffset := caretLine, caretOffset
	for i, sql := range list {
		sqlEndLine := int(sql.End.GetLine()) - 1
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLineZeroBased || (sqlEndLine == caretLineZeroBased && sqlEndColumn >= caretOffset) {
			start = i
			if i == 0 {
				break
			}
			previousEndLine := int(list[i-1].End.GetLine()) - 1
			previousEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLineZeroBased - previousEndLine + 1
			if caretLineZeroBased == previousEndLine {
				newCaretOffset = caretOffset - previousEndColumn + 1
			}
			break
		}
	}

	return list[start].Text, newCaretLine, newCaretOffset
}

func buildCompletionCatalog(ctx context.Context, cCtx base.CompletionContext) *pgcatalog.Catalog {
	if cCtx.Metadata == nil || cCtx.DefaultDatabase == "" {
		return nil
	}

	_, metadata, err := cCtx.Metadata(ctx, cCtx.InstanceID, cCtx.DefaultDatabase)
	if err != nil || metadata == nil {
		return nil
	}

	cat := pgcatalog.New()
	searchPath := []string{"public"}
	if cCtx.DefaultSchema != "" {
		searchPath = []string{cCtx.DefaultSchema, "public"}
	}
	cat.SetSearchPath(searchPath)
	protoMeta := metadata.GetProto()
	_, _ = cat.LoadMetadata(ctx, protoMeta, pgcatalog.LoadMetadataOptions{})
	for _, schema := range protoMeta.GetSchemas() {
		for _, seq := range schema.GetSequences() {
			if seq.GetName() == "" {
				continue
			}
			_ = cat.DefineSequence(&pgast.CreateSeqStmt{
				Sequence: &pgast.RangeVar{
					Schemaname: schema.GetName(),
					Relname:    seq.GetName(),
				},
			})
		}
	}
	return cat
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func completionCandidateText(candidate pgcompletion.Candidate, caretInQuotes bool) string {
	switch candidate.Type {
	case pgcompletion.CandidateKeyword:
		return candidate.Text
	case pgcompletion.CandidateFunction:
		if strings.HasSuffix(candidate.Text, "()") {
			return candidate.Text
		}
		return candidate.Text + "()"
	default:
		if caretInQuotes {
			return candidate.Text
		}
		return quotedIdentifierIfNeeded(candidate.Text)
	}
}

func quotedIdentifierIfNeeded(s string) string {
	if strings.ToLower(s) != s || pgparser.IsReservedKeyword(strings.ToUpper(s)) || !isValidUnquotedIdentifier(s) {
		return quoteIdent(s)
	}
	return s
}

func isValidUnquotedIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	first := rune(s[0])
	if !unicode.IsLetter(first) && first != '_' {
		return false
	}
	for _, ch := range s[1:] {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return false
		}
	}
	return true
}

func caretInsideDoubleQuotes(sql string, byteOffset int) bool {
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}
	inSingle := false
	inDouble := false
	for i := 0; i < byteOffset; i++ {
		switch sql[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		default:
		}
	}
	return inDouble
}

func convertCandidateType(t pgcompletion.CandidateType) base.CandidateType {
	switch t {
	case pgcompletion.CandidateKeyword, pgcompletion.CandidateType_:
		return base.CandidateTypeKeyword
	case pgcompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case pgcompletion.CandidateTable:
		return base.CandidateTypeTable
	case pgcompletion.CandidateView:
		return base.CandidateTypeView
	case pgcompletion.CandidateMaterializedView:
		return base.CandidateTypeMaterializedView
	case pgcompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case pgcompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case pgcompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case pgcompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case pgcompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	default:
		return base.CandidateTypeNone
	}
}

// caretToByteOffset converts a (1-based line, 0-based column) caret position
// into a byte offset into statement. The column is interpreted as a UTF-16
// code-unit offset, matching Monaco/LSP completion positions.
func caretToByteOffset(statement string, caretLine, caretColumn int) int {
	line := 1
	col := 0
	for i := 0; i < len(statement); {
		if line == caretLine && col >= caretColumn {
			return i
		}
		r, size := utf8.DecodeRuneInString(statement[i:])
		if r == '\n' {
			if line == caretLine {
				return i
			}
			line++
			col = 0
		} else if r <= 0xFFFF {
			col++
		} else {
			col += 2
		}
		i += size
	}
	return len(statement)
}
