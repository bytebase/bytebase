package lsp

import lsp "github.com/bytebase/lsp-protocol"

type PingResult struct {
	Result string `json:"result"`
}

type SQLStatementRangesParams struct {
	URI    lsp.DocumentURI `json:"uri"`
	Ranges []lsp.Range     `json:"ranges"`
}
