package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"

	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

// GetFS returns the file system.
func (h *Handler) GetFS() *MemFS {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.fs
}

func (h *Handler) readFile(_ context.Context, uri lsp.DocumentURI) ([]byte, error) {
	if !IsURI(uri) {
		return nil, &os.PathError{Op: "Open", Path: string(uri), Err: errors.New("unable to read out-of-workspace resource from virtual file system")}
	}
	fs := h.GetFS()
	content, found := fs.get(uri)
	if !found {
		return nil, &os.PathError{Op: "Open", Path: string(uri), Err: os.ErrNotExist}
	}
	return content, nil
}

// handleFileSystemRequest handles textDocument/did* requests.
func (h *Handler) handleFileSystemRequest(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (lsp.DocumentURI, bool, error) {
	fs := h.GetFS()

	do := func(uri lsp.DocumentURI, op func() error) (lsp.DocumentURI, bool, error) {
		before, beforeErr := h.readFile(ctx, uri)
		if beforeErr != nil && !os.IsNotExist(beforeErr) {
			// There is no op that could succeed in this case.
			// Most commonly occurs when uri refers to a dir, not a file.
			return uri, false, beforeErr
		}
		err := op()
		after, afterErr := h.readFile(ctx, uri)
		if os.IsNotExist(beforeErr) && os.IsNotExist(afterErr) {
			// File did not exist before or after so nothing has changed.
			return uri, false, err
		} else if afterErr != nil || beforeErr != nil {
			// If an error prevented us from reading the file
			// before or after then we assume the file changed to be conservative.
			return uri, true, err
		}
		return uri, !bytes.Equal(before, after), err
	}

	switch Method(req.Method) {
	case LSPMethodTextDocumentDidOpen:
		var params lsp.DidOpenTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return "", false, err
		}
		return do(params.TextDocument.URI, func() error {
			fs.DidOpen(&params)
			return nil
		})
	case LSPMethodTextDocumentDidChange:
		var params lsp.DidChangeTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return "", false, err
		}
		return do(params.TextDocument.URI, func() error {
			if err := fs.DidChange(&params); err != nil {
				return err
			}
			// Beta: Parse and send diagnostics for MySQL.
			if h.profile.Mode == common.ReleaseModeDev && h.getEngineType(ctx) == store.Engine_MYSQL {
				uri := params.TextDocument.URI
				content, found := fs.get(uri)
				if !found {
					return &os.PathError{Op: "Open", Path: string(uri), Err: os.ErrNotExist}
				}
				err := parseMySQLStatement(string(content))
				diagnostics := collectDiagnostics(err)
				if err := conn.Notify(ctx, string(LSPMethodPublishDiagnostics), &lsp.PublishDiagnosticsParams{
					URI:         uri,
					Diagnostics: diagnostics,
				}); err != nil {
					return err
				}
			}
			return nil
		})
	case LSPMethodTextDocumentDidClose:
		var params lsp.DidCloseTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return "", false, err
		}
		return do(params.TextDocument.URI, func() error {
			fs.DidClose(&params)
			return nil
		})
	case LSPMethodTextDocumentDidSave:
		var params lsp.DidSaveTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return "", false, err
		}
		// no-op
		return params.TextDocument.URI, false, nil
	default:
		return "", false, errors.Errorf("unknown file system request method: %q", req.Method)
	}
}

func parseMySQLStatement(statement string) *base.SyntaxError {
	inputStream := antlr.NewInputStream(statement)
	lexer := parser.NewMySQLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewMySQLParser(stream)
	lexerErrorListener := &base.ParseErrorListener{
		BaseLine: 0,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		BaseLine: 0,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = false

	_ = p.Script()

	if lexerErrorListener.Err != nil {
		return lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return parserErrorListener.Err
	}
	return nil
}

func collectDiagnostics(err *base.SyntaxError) []lsp.Diagnostic {
	if err == nil {
		return []lsp.Diagnostic{}
	}
	syntaxDiagnostic := lsp.Diagnostic{
		Range: lsp.Range{
			Start: lsp.Position{
				// Convert to zero-based.
				Line:      err.Line - 1,
				Character: err.Column,
			},
			End: lsp.Position{
				// Convert to zero-based.
				Line: err.Line - 1,
				// The end position is exclusive.
				Character: err.Column + 1,
			},
		},
		Severity: lsp.Error,
		Source:   "Syntax check",
		Message:  err.Message,
	}

	return []lsp.Diagnostic{syntaxDiagnostic}
}
