// Package transform provides the schema transformation plugin.
package transform

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser"
)

// SchemaTransformer is the interface for schema transformer.
type SchemaTransformer interface {
	Transform(schema string) (string, error)
}

var (
	differMu sync.RWMutex
	differs  = make(map[parser.EngineType]SchemaTransformer)
)

// Register makes a schema transformer available by the provided id.
// If Register is called twice with the same name or if differ is nil,
// it panics.
func Register(engineType parser.EngineType, d SchemaTransformer) {
	if d == nil {
		panic("parser: Register parser is nil")
	}
	differMu.Lock()
	defer differMu.Unlock()
	if _, dup := differs[engineType]; dup {
		panic("parser: Register called twice for schema transformer " + engineType)
	}
	differs[engineType] = d
}

// SchemaTransform returns the transformed schema.
func SchemaTransform(engineType parser.EngineType, schema string) (string, error) {
	differMu.RLock()
	p, ok := differs[engineType]
	differMu.RUnlock()
	if !ok {
		return "", errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.Transform(schema)
}
