package lsp

import (
	"bytes"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sourcegraph/go-lsp"
)

// MemFS is an in-memory file system.
type MemFS struct {
	mu sync.Mutex
	m  map[string][]byte
}

// NewMemFS returns a new in-memory file system.
func NewMemFS() *MemFS {
	return &MemFS{
		m: make(map[string][]byte),
	}
}

// DidOpen notifies the file system that a file was opened.
func (fs *MemFS) DidOpen(params *lsp.DidOpenTextDocumentParams) {
	fs.set(params.TextDocument.URI, []byte(params.TextDocument.Text))
}

// DidChange notifies the file system that a file was changed.
func (fs *MemFS) DidChange(params *lsp.DidChangeTextDocumentParams) error {
	content, found := fs.get(params.TextDocument.URI)
	if !found {
		return errors.Errorf("received textDocument/didChange for unknown file %q", params.TextDocument.URI)
	}

	content, err := applyContentChanges(params.TextDocument.URI, content, params.ContentChanges)
	if err != nil {
		return err
	}

	fs.set(params.TextDocument.URI, content)
	return nil
}

func applyContentChanges(uri lsp.DocumentURI, content []byte, changes []lsp.TextDocumentContentChangeEvent) ([]byte, error) {
	for _, change := range changes {
		if change.Range == nil && change.RangeLength == 0 {
			content = []byte(change.Text) // new full content
			continue
		}
		start, ok, why := offsetForPosition(content, change.Range.Start)
		if !ok {
			return nil, errors.Errorf("received textDocument/didChange for invalid position %q on %q: %s", change.Range.Start, uri, why)
		}
		var end int
		if change.RangeLength != 0 {
			end = start + int(change.RangeLength)
		} else {
			// RangeLength not specified, work it out from Range.End
			end, ok, why = offsetForPosition(content, change.Range.End)
			if !ok {
				return nil, errors.Errorf("received textDocument/didChange for invalid position %q on %q: %s", change.Range.End, uri, why)
			}
		}
		if start < 0 || end > len(content) || start > end {
			return nil, errors.Errorf("received textDocument/didChange for out of range position %q on %q", change.Range, uri)
		}
		// Try avoid doing too many allocations, so use bytes.Buffer
		b := &bytes.Buffer{}
		b.Grow(start + len(change.Text) + len(content) - end)
		b.Write(content[:start])
		b.WriteString(change.Text)
		b.Write(content[end:])
		content = b.Bytes()
	}
	return content, nil
}

// DidClose notifies the file system that a file was closed.
func (fs *MemFS) DidClose(params *lsp.DidCloseTextDocumentParams) {
	fs.del(params.TextDocument.URI)
}

func (fs *MemFS) set(uri lsp.DocumentURI, content []byte) {
	path := uriToMemFSPath(uri)
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.m[path] = content
}

func (fs *MemFS) del(uri lsp.DocumentURI) {
	path := uriToMemFSPath(uri)
	fs.mu.Lock()
	defer fs.mu.Unlock()
	delete(fs.m, path)
}

func (fs *MemFS) get(uri lsp.DocumentURI) ([]byte, bool) {
	path := uriToMemFSPath(uri)
	fs.mu.Lock()
	defer fs.mu.Unlock()
	content, found := fs.m[path]
	return content, found
}

func uriToMemFSPath(uri lsp.DocumentURI) string {
	if IsURI(uri) {
		return strings.TrimPrefix(string(URIToPath(uri)), "/")
	}
	return string(uri)
}

// isFileSystemRequest returns if this is an LSP method whose sole
// purpose is modifying the contents of the overlay file system.
func isFileSystemRequest(method string) bool {
	switch LSPMethod(method) {
	case LSPMethodTextDocumentDidOpen, LSPMethodTextDocumentDidChange, LSPMethodTextDocumentDidClose, LSPMethodTextDocumentDidSave:
		return true
	default:
		return false
	}
}

// Reset resets the file system with lock.
func (fs *MemFS) Reset() {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.m = make(map[string][]byte)
}
