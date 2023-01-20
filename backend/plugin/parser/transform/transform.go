// Package transform provides the schema transformation plugin.
package transform

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser"
)

// SchemaTransformer is the interface for schema transformer.
type SchemaTransformer interface {
	Transform(schema string) (string, error)
}

var (
	transformMu  sync.RWMutex
	transformers = make(map[parser.EngineType]SchemaTransformer)
)

// Register makes a schema transformer available by the provided id.
// If Register is called twice with the same name or if transformer is nil,
// it panics.
func Register(engineType parser.EngineType, t SchemaTransformer) {
	if t == nil {
		panic("parser: Register parser is nil")
	}
	transformMu.Lock()
	defer transformMu.Unlock()
	if _, dup := transformers[engineType]; dup {
		panic("parser: Register called twice for schema transformer " + engineType)
	}
	transformers[engineType] = t
}

// SchemaTransform returns the transformed schema.
func SchemaTransform(engineType parser.EngineType, schema string) (string, error) {
	transformMu.RLock()
	p, ok := transformers[engineType]
	transformMu.RUnlock()
	if !ok {
		return "", errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.Transform(schema)
}
