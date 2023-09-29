// Package differ provides the schema differ plugin.
package differ

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SchemaDiffer is the interface for schema differ.
type SchemaDiffer interface {
	SchemaDiff(oldStmt, newStmt string, ignoreCaseSensitivity bool) (string, error)
}

var (
	differMu sync.RWMutex
	differs  = make(map[storepb.Engine]SchemaDiffer)
)

// Register makes a differ available by the provided id.
// If Register is called twice with the same name or if differ is nil,
// it panics.
func Register(engineType storepb.Engine, d SchemaDiffer) {
	if d == nil {
		panic("parser: Register parser is nil")
	}
	differMu.Lock()
	defer differMu.Unlock()
	if _, dup := differs[engineType]; dup {
		panic(fmt.Sprintf("parser: Register called twice for differ %s", engineType))
	}
	differs[engineType] = d
}

// SchemaDiff returns the schema diff between old and new statements.
func SchemaDiff(engineType storepb.Engine, oldStmt, newStmt string, ignoreCaseSensitive bool) (string, error) {
	differMu.RLock()
	p, ok := differs[engineType]
	differMu.RUnlock()
	if !ok {
		return "", errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.SchemaDiff(oldStmt, newStmt, ignoreCaseSensitive)
}
