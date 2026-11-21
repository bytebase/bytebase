package redshift

import (
	"context"
	"strings"
	"unicode"

	redshiftcatalog "github.com/bytebase/omni/redshift/catalog"
	redshiftcompletion "github.com/bytebase/omni/redshift/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_REDSHIFT, Completion)
}

// Completion returns Redshift auto-completion candidates backed by omni's
// parser-native completer. Bytebase metadata is loaded into an omni Redshift
// catalog so the completer can resolve schemas, relations, columns, sequences,
// and Redshift-specific grammar slots.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	cat := buildCompletionCatalog(ctx, cCtx)

	sql, line, offset := completionStatementAtCaret(statement, caretLine, caretOffset)
	byteOffset := base.CaretToByteOffset(sql, line, offset)
	candidates := redshiftcompletion.Complete(sql, byteOffset, cat)

	return base.DeduplicateCandidates(candidates, func(c redshiftcompletion.Candidate) (base.Candidate, bool) {
		return base.Candidate{
			Text:       completionCandidateText(c),
			Type:       convertCandidateType(c.Type),
			Definition: c.Definition,
			Comment:    c.Comment,
		}, true
	}, func(a, b base.Candidate) int {
		return strings.Compare(a.String(), b.String())
	}), nil
}

// completionStatementAtCaret drops statements before the caret so table refs
// from earlier statements in the same editor buffer cannot leak into completion.
func completionStatementAtCaret(statement string, caretLine int, caretOffset int) (string, int, int) {
	list, err := SplitSQL(statement)
	if err != nil {
		return statement, caretLine, caretOffset
	}
	return base.StatementAtCaret(list, statement, caretLine, caretOffset, true)
}

func buildCompletionCatalog(ctx context.Context, cCtx base.CompletionContext) *redshiftcatalog.Catalog {
	if cCtx.Metadata == nil || cCtx.DefaultDatabase == "" {
		return nil
	}

	_, metadata, err := cCtx.Metadata(ctx, cCtx.InstanceID, cCtx.DefaultDatabase)
	if err != nil || metadata == nil {
		return nil
	}

	cat := redshiftcatalog.New()
	cat.LoadMetadata(metadata.GetProto())

	searchPath := []string{"public"}
	if cCtx.DefaultSchema != "" {
		searchPath = []string{cCtx.DefaultSchema, "public"}
	}
	cat.SetSearchPath(searchPath)
	return cat
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func completionCandidateText(candidate redshiftcompletion.Candidate) string {
	switch candidate.Type {
	case redshiftcompletion.CandidateKeyword, redshiftcompletion.CandidateFunction:
		return candidate.Text
	default:
		return quotedIdentifierIfNeeded(candidate.Text)
	}
}

func quotedIdentifierIfNeeded(s string) string {
	if strings.ToLower(s) != s || !isValidUnquotedIdentifier(s) {
		return quoteIdent(s)
	}
	return s
}

// isValidUnquotedIdentifier checks if the identifier can be used without quotes in Redshift.
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

func convertCandidateType(t redshiftcompletion.CandidateType) base.CandidateType {
	switch t {
	case redshiftcompletion.CandidateKeyword:
		return base.CandidateTypeKeyword
	case redshiftcompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case redshiftcompletion.CandidateTable:
		return base.CandidateTypeTable
	case redshiftcompletion.CandidateView:
		return base.CandidateTypeView
	case redshiftcompletion.CandidateMaterializedView:
		return base.CandidateTypeMaterializedView
	case redshiftcompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case redshiftcompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case redshiftcompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case redshiftcompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case redshiftcompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	default:
		return base.CandidateTypeNone
	}
}
