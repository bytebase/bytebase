package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

type Method string

const (
	LSPMethodPing           Method = "$ping"
	LSPMethodInitialize     Method = "initialize"
	LSPMethodInitialized    Method = "initialized"
	LSPMethodShutdown       Method = "shutdown"
	LSPMethodExit           Method = "exit"
	LSPMethodCancelRequest  Method = "$/cancelRequest"
	LSPMethodSetTrace       Method = "$/setTrace"
	LSPMethodExecuteCommand Method = "workspace/executeCommand"
	LSPMethodCompletion     Method = "textDocument/completion"

	LSPMethodTextDocumentDidOpen   Method = "textDocument/didOpen"
	LSPMethodTextDocumentDidChange Method = "textDocument/didChange"
	LSPMethodTextDocumentDidClose  Method = "textDocument/didClose"
	LSPMethodTextDocumentDidSave   Method = "textDocument/didSave"

	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics.
	LSPMethodPublishDiagnostics Method = "textDocument/publishDiagnostics"

	// Custom Methods.
	// See dollar request: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#dollarRequests.
	LSPCustomMethodSQLStatementRanges Method = "$/textDocument/statementRanges"
)

// NewHandlerWithAuth creates a new Language Server Protocol handler with authentication.
func NewHandlerWithAuth(s *store.Store, profile *config.Profile, iamManager *iam.Manager, user *store.UserMessage, tokenExpiry time.Time) jsonrpc2.Handler {
	handler := &Handler{
		store:                s,
		profile:              profile,
		user:                 user,
		tokenExpiry:          tokenExpiry,
		iamManager:           iamManager,
		diagnosticsDebouncer: NewDiagnosticsDebouncer(500 * time.Millisecond),                            // 500ms debounce
		contentCache:         NewContentCache(100),                                                       // Cache up to 100 documents
		metadataCache:        expirable.NewLRU[string, *model.DatabaseMetadata](128, nil, 5*time.Minute), // Cache up to 128 database metadata with 5min TTL
	}
	return lspHandler{Handler: jsonrpc2.HandlerWithError(handler.handle)}
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

	// Auth-related fields
	user        *store.UserMessage
	tokenExpiry time.Time
	iamManager  *iam.Manager

	shutDown bool
	profile  *config.Profile
	cancelF  sync.Map // map[jsonrpc2.ID]context.CancelFunc

	// Performance optimizations
	diagnosticsDebouncer *DiagnosticsDebouncer
	contentCache         *ContentCache
	metadataCache        *expirable.LRU[string, *model.DatabaseMetadata]
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

func (h *Handler) getDefaultSchema() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.metadata == nil {
		return ""
	}
	return h.metadata.Schema
}

func (h *Handler) getInstanceID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.metadata == nil {
		return ""
	}
	id, err := common.GetInstanceID(h.metadata.InstanceID)
	if err != nil {
		return ""
	}
	return id
}

func (h *Handler) getScene() base.SceneType {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.metadata == nil {
		return base.SceneTypeAll
	}
	switch h.metadata.Scene {
	case "query":
		return base.SceneTypeQuery
	default:
		return base.SceneTypeAll
	}
}

func (h *Handler) getEngineType(ctx context.Context) storepb.Engine {
	instanceID := h.getInstanceID()
	if instanceID == "" {
		return storepb.Engine_ENGINE_UNSPECIFIED
	}

	instance, err := h.store.GetInstance(ctx, &store.FindInstanceMessage{
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
	return instance.Metadata.GetEngine()
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

func (h *Handler) checkTokenExpiry() error {
	if !h.tokenExpiry.IsZero() && time.Now().After(h.tokenExpiry) {
		return errors.New("access token expired, please reconnect")
	}
	return nil
}

func (h *Handler) checkMetadataPermissions(ctx context.Context, metadata SetMetadataCommandArguments) error {
	if h.user == nil {
		return errors.New("user not authenticated")
	}

	// If database is specified, check database schema permission
	if metadata.DatabaseName != "" && metadata.InstanceID != "" {
		// Need to get database to find its project
		instanceID, err := common.GetInstanceID(metadata.InstanceID)
		if err != nil {
			return err
		}

		database, err := h.store.GetDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &metadata.DatabaseName,
		})
		if err != nil {
			return errors.Wrap(err, "failed to get database")
		}
		if database == nil {
			return errors.Errorf("database %q not found", metadata.DatabaseName)
		}

		// Check bb.databases.getSchema permission
		ok, err := h.iamManager.CheckPermission(ctx, iam.PermissionDatabasesGetSchema, h.user, database.ProjectID)
		if err != nil {
			return errors.Wrap(err, "failed to check permission")
		}
		if !ok {
			return errors.Errorf("no permission to access database %q", metadata.DatabaseName)
		}
	} else if metadata.InstanceID != "" && metadata.DatabaseName == "" {
		// If only instance is specified, check instance get permission
		ok, err := h.iamManager.CheckPermission(ctx, iam.PermissionInstancesGet, h.user)
		if err != nil {
			return errors.Wrap(err, "failed to check permission")
		}
		if !ok {
			return errors.Errorf("no permission to access instance %q", metadata.InstanceID)
		}
	}

	return nil
}

func (h *Handler) reset(params *lsp.InitializeParams) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.init = params
	h.fs = NewMemFS()
	// Clear any pending diagnostics on reset
	if h.diagnosticsDebouncer != nil {
		h.diagnosticsDebouncer.Clear()
	}
	return nil
}

func (h *Handler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("panic: %v", r)
			}
			slog.Error("Panic in LSP handler", log.BBError(err), slog.String("method", req.Method), log.BBStack("panic-stack"))
		}
	}()

	// Check token expiry before processing any request
	if err := h.checkTokenExpiry(); err != nil {
		conn.Close()
		return nil, err
	}

	// Handle ping request before checking if the server is initialized.
	if Method(req.Method) == LSPMethodPing {
		return PingResult{Result: "pong"}, nil
	}

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

		return lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: lsp.Incremental,
				CompletionProvider: &lsp.CompletionOptions{
					TriggerCharacters: []string{".", " "},
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
		var params lsp.CancelParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal cancel request")
		}
		var id jsonrpc2.ID
		if v, ok := params.ID.(string); ok {
			id = jsonrpc2.ID{
				Str:      v,
				IsString: true,
			}
		} else if v, ok := params.ID.(float64); ok {
			// handle json number
			id = jsonrpc2.ID{
				Num: uint64(v),
			}
		}
		cancelFAny, loaded := h.cancelF.LoadAndDelete(id)
		if loaded {
			if cancelF, ok := cancelFAny.(context.CancelFunc); ok {
				cancelF()
			}
		}
		return nil, nil
	case LSPMethodSetTrace:
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
			// Check RBAC permissions before setting metadata
			if err := h.checkMetadataPermissions(ctx, setMetadataParams.Arguments[0]); err != nil {
				return nil, &jsonrpc2.Error{
					Code:    jsonrpc2.CodeInvalidRequest,
					Message: fmt.Sprintf("permission denied: %v", err),
				}
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
		childCtx, cancel := context.WithCancel(ctx)
		h.cancelF.Store(req.ID, cancel)
		defer func() {
			cancel()
			h.cancelF.Delete(req.ID)
		}()
		return h.handleTextDocumentCompletion(childCtx, conn, req, params)
	default:
		if isFileSystemRequest(req.Method) {
			_, _, err := h.handleFileSystemRequest(ctx, conn, req)
			return nil, err
		}
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
	}
}
