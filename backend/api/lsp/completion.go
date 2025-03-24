package lsp

import (
	"context"
	"fmt"
	"log/slog"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// 1MB.
	contentLengthLimit = 1024 * 1024
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
	if len(content) > contentLengthLimit {
		// We don't want to parse a huge file.
		return newEmptyCompletionList(), nil
	}
	_, err = offsetForPosition(content, params.Position)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid position %d:%d", params.Position.Line, params.Position.Character)
	}

	defaultDatabase := h.getDefaultDatabase()
	engine := h.getEngineType(ctx)
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE, storepb.Engine_CLICKHOUSE, storepb.Engine_STARROCKS, storepb.Engine_DORIS:
		// Nothing.
	case storepb.Engine_POSTGRES, storepb.Engine_REDSHIFT, storepb.Engine_RISINGWAVE, storepb.Engine_COCKROACHDB:
		// Nothing.
	case storepb.Engine_MSSQL:
	case storepb.Engine_ORACLE, storepb.Engine_DM, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_SNOWFLAKE:
	case storepb.Engine_DYNAMODB:
	default:
		slog.Debug("Engine is not supported", slog.String("engine", engine.String()))
		return newEmptyCompletionList(), nil
	}
	candidates, err := base.Completion(ctx, engine, base.CompletionContext{
		Scene:             h.getScene(),
		InstanceID:        h.getInstanceID(),
		DefaultDatabase:   defaultDatabase,
		DefaultSchema:     h.getDefaultSchema(),
		Metadata:          h.GetDatabaseMetadataFunc,
		ListDatabaseNames: h.ListDatabaseNamesFunc,
	}, string(content), int(params.Position.Line)+1, int(params.Position.Character))
	if err != nil {
		// return errors will close the websocket connection, so we just log the error and return empty completion list.
		slog.Error("Failed to get completion candidates", "err", err)
		return newEmptyCompletionList(), nil
	}

	var items []lsp.CompletionItem
	for _, candidate := range candidates {
		label := candidate.Text
		// Remove quotes or brackets from label.
		if len(label) > 1 && (label[0] == '"' && label[len(label)-1] == '"' || label[0] == '`' && label[len(label)-1] == '`' || label[0] == '[' && label[len(label)-1] == ']') {
			label = label[1 : len(label)-1]
		}
		completionItem := lsp.CompletionItem{
			Label: label,
			LabelDetails: &lsp.CompletionItemLabelDetails{
				Detail:      fmt.Sprintf("(%s)", string(candidate.Type)),
				Description: candidate.Definition,
			},
			Kind: convertLSPCompletionItemKind(candidate.Type),
			Documentation: &lsp.Or_CompletionItem_documentation{
				Value: candidate.Comment,
			},
			SortText:   generateSortText(params, engine, candidate),
			InsertText: candidate.Text,
		}
		items = append(items, completionItem)
	}

	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func generateSortText(_ lsp.CompletionParams, _ storepb.Engine, candidate base.Candidate) string {
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
	case base.CandidateTypeKeyword:
		switch candidate.Text {
		case "SELECT", "SHOW", "SET", "FROM", "WHERE":
			return "09" + candidate.Text
		default:
			return "10" + candidate.Text
		}
	default:
		return "10" + string(candidate.Type) + candidate.Text
	}
}

func convertLSPCompletionItemKind(tp base.CandidateType) lsp.CompletionItemKind {
	switch tp {
	case base.CandidateTypeSchema:
		return lsp.ModuleCompletion
	case base.CandidateTypeDatabase:
		return lsp.ClassCompletion
	case base.CandidateTypeTable, base.CandidateTypeForeignTable:
		return lsp.FieldCompletion
	case base.CandidateTypeColumn:
		return lsp.InterfaceCompletion
	case base.CandidateTypeFunction:
		return lsp.FunctionCompletion
	case base.CandidateTypeView, base.CandidateTypeMaterializedView:
		return lsp.VariableCompletion
	default:
		return lsp.TextCompletion
	}
}

func (h *Handler) GetDatabaseMetadataFunc(ctx context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
	// TODO: do ACL check here.
	if instanceID == "" {
		return "", nil, errors.Errorf("instance is not specified")
	}

	database, err := h.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to get database")
	}
	if database == nil {
		return "", nil, errors.Errorf("database %s for instance %s not found", databaseName, instanceID)
	}
	metadata, err := h.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to get database schema")
	}
	if metadata == nil {
		return "", nil, errors.Errorf("database %s schema for instance %s not found", databaseName, instanceID)
	}
	return databaseName, metadata.GetDatabaseMetadata(), nil
}

func (h *Handler) ListDatabaseNamesFunc(ctx context.Context, instanceID string) ([]string, error) {
	if instanceID == "" {
		return nil, errors.Errorf("instance is not specified")
	}

	databases, err := h.store.ListDatabases(ctx, &store.FindDatabaseMessage{
		InstanceID: &instanceID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list databases")
	}
	var names []string
	for _, database := range databases {
		names = append(names, database.DatabaseName)
	}
	return names, nil
}
