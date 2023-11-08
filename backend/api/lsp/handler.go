package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/pkg/errors"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type Method string

const (
	LSPMethodInitialize     Method = "initialize"
	LSPMethodInitialized    Method = "initialized"
	LSPMethodShutdown       Method = "shutdown"
	LSPMethodExit           Method = "exit"
	LSPMethodCancelRequest  Method = "$/cancelRequest"
	LSPMethodExecuteCommand Method = "workspace/executeCommand"
	LSPMethodCompletion     Method = "textDocument/completion"

	LSPMethodTextDocumentDidOpen   Method = "textDocument/didOpen"
	LSPMethodTextDocumentDidChange Method = "textDocument/didChange"
	LSPMethodTextDocumentDidClose  Method = "textDocument/didClose"
	LSPMethodTextDocumentDidSave   Method = "textDocument/didSave"
)

// NewHandler creates a new Language Server Protocol handler.
func NewHandler(s *store.Store) jsonrpc2.Handler {
	return lspHandler{jsonrpc2.HandlerWithError((&Handler{store: s}).handle)}
}

type lspHandler struct {
	jsonrpc2.Handler
}

func (h lspHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if isFileSystemRequest(req.Method) {
		h.Handler.Handle(ctx, conn, req)
		return
	}
	go h.Handler.Handle(ctx, conn, req)
}

// Handler handles Language Server Protocol requests.
type Handler struct {
	mu       sync.Mutex
	fs       *MemFS
	init     *lsp.InitializeParams // set by LSPMethodInitialize request
	metadata *SetMetadataCommandArguments
	store    *store.Store

	shutDown bool
}

// ShutDown shuts down the handler.
func (h *Handler) ShutDown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.shutDown {
		slog.Warn("server received a shutdown request after it was already shut down.")
	}
	h.shutDown = true
	h.fs = nil
}

func (h *Handler) setMetadata(arg SetMetadataCommandArguments) {
	h.mu.Lock()
	defer h.mu.Unlock()
	tmp := arg
	h.metadata = &tmp
}

func (h *Handler) getDefaultDatabase() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.metadata == nil {
		return ""
	}
	return h.metadata.DatabaseName
}

func (h *Handler) getInstanceID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.metadata == nil {
		return ""
	}
	return h.metadata.InstanceID
}

func (h *Handler) getEngineType(ctx context.Context) storepb.Engine {
	instanceID := h.getInstanceID()
	if instanceID == "" {
		return storepb.Engine_ENGINE_UNSPECIFIED
	}

	instance, err := h.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		slog.Error("Failed to get instance", log.BBError(err))
		return storepb.Engine_ENGINE_UNSPECIFIED
	}
	if instance == nil {
		slog.Error("Instance not found", slog.String("instanceID", instanceID))
		return storepb.Engine_ENGINE_UNSPECIFIED
	}
	return instance.Engine
}

func (h *Handler) checkInitialized(req *jsonrpc2.Request) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if Method(req.Method) != LSPMethodInitialize && h.init == nil {
		return errors.New("server must be initialized first")
	}
	return nil
}

func (h *Handler) checkReady() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.shutDown {
		return errors.New("server is shutting down")
	}
	return nil
}

func (h *Handler) reset(params *lsp.InitializeParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.init = params
	h.fs = NewMemFS()
	return nil
}

func (h *Handler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	if err := h.checkInitialized(req); err != nil {
		return nil, err
	}
	if err := h.checkReady(); err != nil {
		return nil, err
	}

	switch Method(req.Method) {
	case LSPMethodInitialize:
		if h.init != nil {
			return nil, errors.New("server is already initialized")
		}
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.InitializeParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		if err := h.reset(&params); err != nil {
			return nil, err
		}

		kind := lsp.TDSKIncremental
		return lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
					Kind: &kind,
				},
				CompletionProvider: &lsp.CompletionOptions{
					TriggerCharacters: []string{"."},
				},
				ExecuteCommandProvider: &lsp.ExecuteCommandOptions{
					Commands: []string{string(CommandNameSetMetadata)},
				},
			},
		}, nil
	case LSPMethodInitialized:
		// A notification that the client is ready to receive requests. Ignore.
		return nil, nil
	case LSPMethodShutdown:
		h.ShutDown()
		return nil, nil
	case LSPMethodExit:
		conn.Close()
		h.ShutDown()
		return nil, nil
	case LSPMethodCancelRequest:
		// Do nothing for now.
		return nil, nil
	case LSPMethodExecuteCommand:
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.ExecuteCommandParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		switch CommandName(params.Command) {
		case CommandNameSetMetadata:
			var setMetadataParams SetMetadataCommandParams
			if err := json.Unmarshal(*req.Params, &setMetadataParams); err != nil {
				return nil, err
			}
			if len(setMetadataParams.Arguments) != 1 {
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: "expected exactly one argument"}
			}
			h.setMetadata(setMetadataParams.Arguments[0])
			return nil, nil
		default:
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams, Message: fmt.Sprintf("command not supported: %s", params.Command)}
		}
	case LSPMethodCompletion:
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var params lsp.CompletionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return h.handleTextDocumentCompletion(ctx, conn, req, params)
	default:
		if isFileSystemRequest(req.Method) {
			_, _, err := h.handleFileSystemRequest(ctx, req)
			return nil, err
		}
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
	}
}
