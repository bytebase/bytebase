package lsp

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sourcegraph/go-lsp"
)

func trimFilePrefix(s string) string {
	return strings.TrimPrefix(s, "file://")
}

// IsURI returns true if the given URI is a file:// URI.
func IsURI(uri lsp.DocumentURI) bool {
	return strings.HasPrefix(string(uri), "file://")
}

// URIToPath converts a URI to a file path.
func URIToPath(uri lsp.DocumentURI) string {
	u, err := url.Parse(string(uri))
	if err != nil {
		return trimFilePrefix(string(uri))
	}
	return u.Path
}

func offsetForPosition(content []byte, p lsp.Position) (offset int, valid bool, whyInvalid string) {
	line := 0
	col := 0
	for _, b := range content {
		if line == p.Line && col == p.Character {
			return offset, true, ""
		}
		if (line == p.Line && col > p.Character) || line > p.Line {
			return 0, false, fmt.Sprintf("character %d (zero-based) is beyond line %d boundary (zero-based)", p.Character, p.Line)
		}
		offset++
		if b == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	if line == p.Line && col == p.Character {
		return offset, true, ""
	}
	if line == 0 {
		return 0, false, fmt.Sprintf("character %d (zero-based) is beyond first line boundary", p.Character)
	}
	return 0, false, fmt.Sprintf("file only has %d lines", line+1)
}
