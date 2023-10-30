package lsp

import (
	"context"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

// NewHandler creates a new Language Server Protocol handler.
func NewHandler() jsonrpc2.Handler {
	return lspHandler{jsonrpc2.HandlerWithError((&Handler{}).handle)}
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

// isFileSystemRequest returns if this is an LSP method whose sole
// purpose is modifying the contents of the overlay file system.
func isFileSystemRequest(method string) bool {
	return method == "textDocument/didOpen" ||
		method == "textDocument/didChange" ||
		method == "textDocument/didClose" ||
		method == "textDocument/didSave"
}

// Handler handles Language Server Protocol requests.
type Handler struct {
}

func (*Handler) handle(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
	// TODO: implement
	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}
