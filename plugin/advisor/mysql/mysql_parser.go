package mysql

import "github.com/pingcap/parser"

// Wrapper for parser.New().
func newParser() *parser.Parser {
	p := parser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	return p
}
