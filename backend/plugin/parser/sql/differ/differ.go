// Package differ provides the schema differ plugin.
package differ

import (
	"sync"

	"github.com/pkg/errors"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

// SchemaDiffer is the interface for schema differ.
type SchemaDiffer interface {
	SchemaDiff(oldStmt, newStmt string, ignoreCaseSensitivity bool) (string, error)
}

var (
	differMu sync.RWMutex
	differs  = make(map[parser.EngineType]SchemaDiffer)
)

// Register makes a differ available by the provided id.
// If Register is called twice with the same name or if differ is nil,
// it panics.
func Register(engineType parser.EngineType, d SchemaDiffer) {
	if d == nil {
		panic("parser: Register parser is nil")
	}
	differMu.Lock()
	defer differMu.Unlock()
	if _, dup := differs[engineType]; dup {
		panic("parser: Register called twice for differ " + engineType)
	}
	differs[engineType] = d
}

// SchemaDiff returns the schema diff between old and new statements.
func SchemaDiff(engineType parser.EngineType, oldStmt, newStmt string, ignoreCaseSensitive bool) (string, error) {
	differMu.RLock()
	p, ok := differs[engineType]
	differMu.RUnlock()
	if !ok {
		return "", errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.SchemaDiff(oldStmt, newStmt, ignoreCaseSensitive)
}
