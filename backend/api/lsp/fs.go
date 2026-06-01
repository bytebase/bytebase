package lsp

import (
	"bytes"
	"strings"
	"sync"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/pkg/errors"
)

// openDocument is a document tracked by the in-memory file system. The
// original URI is retained alongside the content so callers can enumerate open
// documents (the map is keyed by mem-FS path, which is not reversible to a URI
// in the general case).
type openDocument struct {
	uri     lsp.DocumentURI
	content []byte
}

// MemFS is an in-memory file system.
type MemFS struct {
	mu sync.Mutex
	m  map[string]openDocument
}

// NewMemFS returns a new in-memory file system.
func NewMemFS() *MemFS {
	return &MemFS{
		m: make(map[string]openDocument),
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
		start, err := offsetForPosition(content, change.Range.Start)
		if err != nil {
			return nil, errors.Wrapf(err, "received textDocument/didChange for invalid position %v on %q", change.Range.Start, uri)
		}

		end, err := offsetForPosition(content, change.Range.End)
		if err != nil {
			return nil, errors.Wrapf(err, "received textDocument/didChange for invalid position %v on %q", change.Range.End, uri)
		}
		if start < 0 || end > len(content) || start > end {
			return nil, errors.Errorf("received textDocument/didChange for out of range position %v on %q", change.Range, uri)
		}

		// Try avoid doing too many allocations, so use bytes.Buffer
		b := &bytes.Buffer{}
		b.Grow(start + len(change.Text) + len(content) - end)
		if _, err := b.Write(content[:start]); err != nil {
			return nil, err
		}
		if _, err := b.WriteString(change.Text); err != nil {
			return nil, err
		}
		if _, err := b.Write(content[end:]); err != nil {
			return nil, err
		}
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
	fs.m[path] = openDocument{uri: uri, content: content}
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
	doc, found := fs.m[path]
	return doc.content, found
}

// listOpen returns a snapshot of all currently open documents.
func (fs *MemFS) listOpen() []openDocument {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	docs := make([]openDocument, 0, len(fs.m))
	for _, doc := range fs.m {
		docs = append(docs, doc)
	}
	return docs
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
	switch Method(method) {
	case LSPMethodTextDocumentDidOpen, LSPMethodTextDocumentDidChange, LSPMethodTextDocumentDidClose, LSPMethodTextDocumentDidSave:
		return true
	default:
		return false
	}
}
