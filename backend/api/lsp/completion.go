package lsp

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/pkg/errors"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"

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
	if engine == storepb.Engine_ENGINE_UNSPECIFIED {
		// return errors will close the websocket connection, so we just log the error and return empty completion list.
		slog.Error("Engine is not specified")
		return newEmptyCompletionList(), nil
	}
	candidates, err := base.Completion(ctx, engine, string(content), params.Position.Line+1, params.Position.Character, defaultDatabase, h.GetDatabaseMetadataFunc)
	if err != nil {
		// return errors will close the websocket connection, so we just log the error and return empty completion list.
		slog.Error("Failed to get completion candidates", "err", err)
		return newEmptyCompletionList(), nil
	}

	candidates = sortCandidates(candidates, params)

	var items []lsp.CompletionItem
	for _, candidate := range candidates {
		items = append(items, lsp.CompletionItem{
			Label:  candidate.Text,
			Detail: fmt.Sprintf("<%s>", string(candidate.Type)),
			Kind:   convertLSPCompletionItemKind(candidate.Type),
		})
	}

	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func sortCandidates(candidates []base.Candidate, params lsp.CompletionParams) []base.Candidate {
	switch params.Context.TriggerCharacter {
	case ".":
		sort.Slice(candidates, func(i, j int) bool {
			priorityI := candidateTypePriorityAfterDot(candidates[i].Type)
			priorityJ := candidateTypePriorityAfterDot(candidates[j].Type)
			if priorityI != priorityJ {
				return priorityI < priorityJ
			}
			if candidates[i].Type != candidates[j].Type {
				return candidates[i].Type < candidates[j].Type
			}
			return candidates[i].Text < candidates[j].Text
		})
	}

	return candidates
}

func candidateTypePriorityAfterDot(tp base.CandidateType) int {
	switch tp {
	case base.CandidateTypeSchema:
		return 1
	case base.CandidateTypeTable:
		return 2
	case base.CandidateTypeView:
		return 3
	case base.CandidateTypeColumn:
		return 4
	case base.CandidateTypeFunction:
		return 5
	default:
		return 6
	}
}

func convertLSPCompletionItemKind(tp base.CandidateType) lsp.CompletionItemKind {
	switch tp {
	case base.CandidateTypeDatabase:
		return lsp.CIKClass
	case base.CandidateTypeTable:
		return lsp.CIKField
	case base.CandidateTypeColumn:
		return lsp.CIKInterface
	case base.CandidateTypeFunction:
		return lsp.CIKFunction
	case base.CandidateTypeView:
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
