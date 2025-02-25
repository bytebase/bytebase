package lsp

import "github.com/sourcegraph/go-lsp"

type PingResult struct {
	Result string `json:"result"`
}

type SQLStatementRangesParams struct {
	URI    lsp.DocumentURI `json:"uri"`
	Ranges []lsp.Range     `json:"ranges"`
}
