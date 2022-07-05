package parser

import (
	"fmt"
	"sync"

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

// Context is the context for parser.
type Context struct {
}

// Parser is the interface for parser.
type Parser interface {
	Parse(ctx Context, statement string) ([]ast.Node, error)
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
func Parse(engineType EngineType, ctx Context, statement string) ([]ast.Node, error) {
	parserMu.RLock()
	p, ok := parsers[engineType]
	parserMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.Parse(ctx, statement)
}
