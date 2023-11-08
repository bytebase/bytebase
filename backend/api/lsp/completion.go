package lsp

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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
	if engine == storepb.Engine_ENGINE_UNSPECIFIED {
		return nil, errors.Errorf("engine is not specified")
	}
	candidates, err := base.Completion(ctx, engine, string(content), params.Position.Line+1, params.Position.Character, defaultDatabase, h.GetDatabaseMetadataFunc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get completion candidates")
	}

	var items []lsp.CompletionItem
	for _, candidate := range candidates {
		items = append(items, lsp.CompletionItem{
			Label:  candidate.Text,
			Detail: fmt.Sprintf("<%s>", string(candidate.Type)),
			Kind:   lsp.CIKKeyword,
		})
	}

	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
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
