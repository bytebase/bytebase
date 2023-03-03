// Package transform provides the schema transformation plugin.
package transform

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser"
)

// SchemaTransformer is the interface for schema SDL format transformer.
type SchemaTransformer interface {
	Transform(schema string) (string, error)
	Check(schema string) (int, error)
	Normalize(schema string, standard string) (string, error)
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

// SchemaTransform returns the transformed schema(SDL format).
func SchemaTransform(engineType parser.EngineType, schema string) (string, error) {
	transformMu.RLock()
	p, ok := transformers[engineType]
	transformMu.RUnlock()
	if !ok {
		return "", errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.Transform(schema)
}

// CheckFormat checks the schema format.
func CheckFormat(engineType parser.EngineType, schema string) (int, error) {
	transformMu.RLock()
	p, ok := transformers[engineType]
	transformMu.RUnlock()
	if !ok {
		return 0, errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.Check(schema)
}

// Normalize normalizes the schema format. The schema and standard should be SDL format.
func Normalize(engineType parser.EngineType, schema string, standard string) (string, error) {
	transformMu.RLock()
	p, ok := transformers[engineType]
	transformMu.RUnlock()
	if !ok {
		return "", errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return p.Normalize(schema, standard)
}
