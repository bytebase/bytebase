package lsp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
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
	if !common.EngineSupportAutoComplete(engine) {
		slog.Debug("Engine is not supported", slog.String("engine", engine.String()))
		return newEmptyCompletionList(), nil
	}
	candidates, err := parserbase.Completion(ctx, engine, parserbase.CompletionContext{
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

	items := []lsp.CompletionItem{}
	for _, candidate := range candidates {
		label := candidate.Text
		// Remove quotes or brackets from label.
		if len(label) > 1 && (label[0] == '"' && label[len(label)-1] == '"' || label[0] == '`' && label[len(label)-1] == '`' || label[0] == '[' && label[len(label)-1] == ']') {
			label = label[1 : len(label)-1]
			label = strings.ReplaceAll(label, `""`, `"`)
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

func generateSortText(_ lsp.CompletionParams, _ storepb.Engine, candidate parserbase.Candidate) string {
	switch candidate.Type {
	case parserbase.CandidateTypeColumn:
		return "01" + candidate.TextWithPriority()
	case parserbase.CandidateTypeSchema:
		return "02" + candidate.TextWithPriority()
	case parserbase.CandidateTypeTable, parserbase.CandidateTypeForeignTable:
		return "03" + candidate.TextWithPriority()
	case parserbase.CandidateTypeView, parserbase.CandidateTypeMaterializedView, parserbase.CandidateTypeSequence:
		return "04" + candidate.TextWithPriority()
	case parserbase.CandidateTypeFunction:
		return "05" + candidate.TextWithPriority()
	case parserbase.CandidateTypeKeyword:
		switch candidate.Text {
		case "SELECT", "SHOW", "SET", "FROM", "WHERE":
			return "09" + candidate.TextWithPriority()
		default:
			return "10" + candidate.TextWithPriority()
		}
	default:
		return "10" + string(candidate.Type) + candidate.TextWithPriority()
	}
}

func convertLSPCompletionItemKind(tp parserbase.CandidateType) lsp.CompletionItemKind {
	switch tp {
	case parserbase.CandidateTypeSchema:
		return lsp.ModuleCompletion
	case parserbase.CandidateTypeDatabase:
		return lsp.ClassCompletion
	case parserbase.CandidateTypeTable, parserbase.CandidateTypeForeignTable:
		return lsp.FieldCompletion
	case parserbase.CandidateTypeColumn:
		return lsp.InterfaceCompletion
	case parserbase.CandidateTypeFunction:
		return lsp.FunctionCompletion
	case parserbase.CandidateTypeView, parserbase.CandidateTypeMaterializedView:
		return lsp.VariableCompletion
	case parserbase.CandidateTypeSequence:
		return lsp.ConstantCompletion
	default:
		return lsp.TextCompletion
	}
}

func (h *Handler) GetDatabaseMetadataFunc(ctx context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
	// Check cache first
	cacheKey := getDatabaseMetadataCacheKey(instanceID, databaseName)
	if cached, exists := h.metadataCache.Get(cacheKey); exists {
		return databaseName, cached, nil
	}

	// Cache miss, fetch from store
	metadata, err := h.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
	})
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to get database schema")
	}
	if metadata == nil {
		return "", nil, errors.Errorf("database %s schema for instance %s not found", databaseName, instanceID)
	}

	// Store in cache
	h.metadataCache.Add(cacheKey, metadata)

	return databaseName, metadata, nil
}

func getDatabaseMetadataCacheKey(instanceID, databaseName string) string {
	return instanceID + "/" + databaseName
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
