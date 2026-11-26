// Package trino provides a SQL parser adapter for Trino, backed by the omni
// hand-written Trino parser (github.com/bytebase/omni/trino/*).
package trino

import (
	"context"
	stderrors "errors"

	"github.com/bytebase/omni/trino/analysis"
	"github.com/bytebase/omni/trino/completion"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine)
	engine.Complete = Completion
	engine.QuerySpan = GetQuerySpan
	base.Register(engine, storepb.Engine_TRINO)
}

// Completion returns auto-completion candidates for a Trino statement at the
// given caret position, backed by omni's session-aware Trino completer.
// Bytebase metadata (database -> schema -> table -> column) maps onto the Trino
// catalog model as catalog=database.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadCatalog, listCatalogs := base.CompletionMetadataLoaders(cCtx)
	cands := completion.CompleteSession(ctx, completion.SessionConfig{
		DefaultCatalog:   cCtx.DefaultDatabase,
		DefaultSchema:    cCtx.DefaultSchema,
		LoadCatalog:      loadCatalog,
		ListCatalogNames: listCatalogs,
	}, statement, caretLine, caretOffset)

	result := make([]base.Candidate, 0, len(cands))
	for _, c := range cands {
		result = append(result, base.Candidate{
			Text: c.Text,
			Type: base.OmniCandidateType(c.Type),
		})
	}
	return result, nil
}

// GetQuerySpan gets the query span for a Trino statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, ignoreCaseSensitive bool) (*base.QuerySpan, error) {
	resolved, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		ListDatabases:       base.QuerySpanListDatabasesFunc(ctx, gCtx, false),
		GetDatabaseMetadata: base.QuerySpanProtoMetadataFunc(ctx, gCtx, true),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	span := base.OmniQuerySpan(resolved.Type, resolved.SourceColumns, resolved.PredicateColumns, resolved.Results, nil)
	if resolved.NotFoundError != nil {
		converted := base.OmniNotFoundError(resolved.NotFoundError)
		var nf *base.ResourceNotFoundError
		if stderrors.As(converted, &nf) {
			span.NotFoundError = nf
		} else {
			span.NotFoundError = &base.ResourceNotFoundError{Err: resolved.NotFoundError}
		}
	}
	return span, nil
}
