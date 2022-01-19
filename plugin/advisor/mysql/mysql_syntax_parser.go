package mysql

import "github.com/pingcap/parser"

// Wrapper for parser.New().
func newParser() *parser.Parser {
	p := parser.New()
	p.EnableWindowFunc(true)
	return p
}
