package lsp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func newEmptyCompletionList() *lsp.CompletionList {
	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        []lsp.CompletionItem{},
	}
}

func (h *Handler) handleTextDocumentCompletion(ctx context.Context, _ *jsonrpc2.Conn, _ *jsonrpc2.Request, params lsp.CompletionParams) (*lsp.CompletionList, error) {
	if !IsURI(params.TextDocument.URI) {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("textDocument/completion not yet supported for out-of-workspace URI (%q)", params.TextDocument.URI),
		}
	}
	content, err := h.readFile(ctx, params.TextDocument.URI)
	if err != nil {
		return nil, err
	}
	_, valid, why := offsetForPosition(content, params.Position)
	if !valid {
		return nil, errors.Errorf("invalid position %d:%d (%s)", params.Position.Line, params.Position.Character, why)
	}

	defaultDatabase := h.getDefaultDatabase()
	engine := h.getEngineType(ctx)
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE, storepb.Engine_CLICKHOUSE, storepb.Engine_STARROCKS, storepb.Engine_DORIS:
		// Nothing.
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_RISINGWAVE:
		// Nothing.
	case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_SNOWFLAKE, storepb.Engine_MSSQL:
	default:
		slog.Debug("Engine is not supported", slog.String("engine", engine.String()))
		return newEmptyCompletionList(), nil
	}
	candidates, err := base.Completion(ctx, engine, string(content), params.Position.Line+1, params.Position.Character, defaultDatabase, h.GetDatabaseMetadataFunc)
	if err != nil {
		// return errors will close the websocket connection, so we just log the error and return empty completion list.
		slog.Error("Failed to get completion candidates", "err", err)
		return newEmptyCompletionList(), nil
	}

	var items []lsp.CompletionItem
	for _, candidate := range candidates {
		completionItem := lsp.CompletionItem{
			Label: candidate.Text,
			LabelDetails: &lsp.CompletionItemLabelDetails{
				Detail:      fmt.Sprintf("(%s)", string(candidate.Type)),
				Description: candidate.Definition,
			},
			Kind:          convertLSPCompletionItemKind(candidate.Type),
			Documentation: candidate.Comment,
			SortText:      generateSortText(params, candidate),
			InsertText:    generateInsertText(engine, candidate),
		}
		items = append(items, completionItem)
	}

	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func generateSortText(params lsp.CompletionParams, candidate base.Candidate) string {
	switch params.Context.TriggerCharacter {
	case ".":
		return generateSortTextAfterDot(candidate)
	default:
		return string(candidate.Type) + candidate.Text
	}
}

func generateSortTextAfterDot(candidate base.Candidate) string {
	switch candidate.Type {
	case base.CandidateTypeColumn:
		return "01" + candidate.Text
	case base.CandidateTypeSchema:
		return "02" + candidate.Text
	case base.CandidateTypeTable, base.CandidateTypeForeignTable:
		return "03" + candidate.Text
	case base.CandidateTypeView, base.CandidateTypeMaterializedView:
		return "04" + candidate.Text
	case base.CandidateTypeFunction:
		return "05" + candidate.Text
	default:
		return "10" + string(candidate.Type) + candidate.Text
	}
}

func generateInsertText(engine storepb.Engine, candidate base.Candidate) string {
	// For non-postgres engine, we return the candidate text as the insert text.
	if engine != storepb.Engine_POSTGRES {
		return candidate.Text
	}

	insertText := candidate.Text
	// If the candidate is a column and it's a camel case, we need to quote it.
	//
	// TODO(rebelice): there are lots of special cases that we don't need to add quotes.
	// For example, if the previous token is a quote, we don't need to quote the column name.
	if candidate.Type == base.CandidateTypeColumn && common.IsCamelCase(insertText) {
		insertText = fmt.Sprintf(`"%s"`, insertText)
	}
	return insertText
}

func convertLSPCompletionItemKind(tp base.CandidateType) lsp.CompletionItemKind {
	switch tp {
	case base.CandidateTypeDatabase:
		return lsp.CIKClass
	case base.CandidateTypeTable, base.CandidateTypeForeignTable:
		return lsp.CIKField
	case base.CandidateTypeColumn:
		return lsp.CIKInterface
	case base.CandidateTypeFunction:
		return lsp.CIKFunction
	case base.CandidateTypeView, base.CandidateTypeMaterializedView:
		return lsp.CIKVariable
	default:
		return lsp.CIKText
	}
}

func (h *Handler) GetDatabaseMetadataFunc(ctx context.Context, databaseName string) (*model.DatabaseMetadata, error) {
	// TODO: do ACL check here.
	instanceID := h.getInstanceID()
	if instanceID == "" {
		return nil, errors.Errorf("instance is not specified")
	}

	database, err := h.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database %s for instance %s not found", databaseName, instanceID)
	}
	metadata, err := h.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database schema")
	}
	if metadata == nil {
		return nil, errors.Errorf("database %s schema for instance %s not found", databaseName, instanceID)
	}
	return metadata.GetDatabaseMetadata(), nil
}
