// Package parser provides the interfaces and libraries for SQL parser.
package parser

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser/ast"
)

// EngineType is the type of a parser engine.
type EngineType string

const (
	// MySQL is the engine type for MYSQL.
	MySQL EngineType = "MYSQL"
	// Postgres is the engine type for POSTGRES.
	Postgres EngineType = "POSTGRES"
	// TiDB is the engine type for TiDB.
	TiDB EngineType = "TIDB"
)

// ParseContext is the context for parsing.
type ParseContext struct {
}

// DeparseContext is the contxt for restoring.
type DeparseContext struct {
	// Indent is indent at the beginning of the line.
	// The parser deparses statements with the indent for pretty format.
	Indent string
}

// Parser is the interface for parser.
type Parser interface {
	Parse(ctx ParseContext, statement string) ([]ast.Node, error)
	Deparse(ctx DeparseContext, node ast.Node) (string, error)
}

var (
	parserMu sync.RWMutex
	parsers  = make(map[EngineType]Parser)
)

// Register makes a parser available by the provided id.
// If Register is called twice with the same name or if advisor is nil,
// it panics.
func Register(engineType EngineType, p Parser) {
	if p == nil {
		panic("parser: Register parser is nil")
	}
	parserMu.Lock()
	defer parserMu.Unlock()
	if _, dup := parsers[engineType]; dup {
		panic("parser: Register called twice for parser " + engineType)
	}
	parsers[engineType] = p
}

// Parse parses the statement and return nodes.
func Parse(engineType EngineType, ctx ParseContext, statement string) ([]ast.Node, error) {
	parserMu.RLock()
	p, ok := parsers[engineType]
	parserMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.Parse(ctx, statement)
}

// Deparse deparses the statement from node(AST).
func Deparse(engineType EngineType, ctx DeparseContext, node ast.Node) (string, error) {
	parserMu.RLock()
	p, ok := parsers[engineType]
	parserMu.RUnlock()
	if !ok {
		return "", errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.Deparse(ctx, node)
}
