package pg

import (
	"context"
	"strings"
	"unicode"

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
	byteOffset := base.CaretToByteOffset(sql, line, offset)
	caretInQuotes := caretInsideDoubleQuotes(sql, byteOffset)
	candidates := pgcompletion.Complete(sql, byteOffset, cat)

	return base.DeduplicateCandidates(candidates, func(c pgcompletion.Candidate) (base.Candidate, bool) {
		return base.Candidate{
			Text:       completionCandidateText(c, caretInQuotes),
			Type:       convertCandidateType(c.Type),
			Definition: c.Definition,
			Comment:    c.Comment,
		}, true
	}, base.CompareCandidateByTypeText), nil
}

// completionStatementAtCaret isolates the statement containing the caret so
// neither table refs from earlier statements nor broken trailing statements in
// the same editor buffer affect completion.
func completionStatementAtCaret(statement string, caretLine int, caretOffset int) (string, int, int) {
	list, err := SplitSQL(statement)
	if err != nil {
		return statement, caretLine, caretOffset
	}
	return base.StatementAtCaret(list, statement, caretLine, caretOffset, false)
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
	if byteOffset <= 0 {
		return false
	}
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}
	lex := pgparser.NewLexerRange(sql, 0, byteOffset)
	for {
		tok := lex.NextToken()
		if tok.Type == 0 {
			break
		}
	}
	return lex.Err != nil && lex.Err.Error() == "unterminated quoted identifier"
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
