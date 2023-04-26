// Package edit provides the schema edit plugin.
package edit

import (
	"sync"

	"github.com/pkg/errors"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

// SchemaEditor is the interface for schema editor.
type SchemaEditor interface {
	DeparseDatabaseEdit(databaseEdit *api.DatabaseEdit) (string, error)
	ValidateDatabaseEdit(databaseEdit *api.DatabaseEdit) ([]*api.ValidateResult, error)
}

var (
	editorMu sync.RWMutex
	editors  = make(map[parser.EngineType]SchemaEditor)
)

// Register makes a differ available by the provided id.
// If Register is called twice with the same name or if differ is nil,
// it panics.
func Register(engineType parser.EngineType, se SchemaEditor) {
	if se == nil {
		panic("parser: Register parser is nil")
	}
	editorMu.Lock()
	defer editorMu.Unlock()
	if _, dup := editors[engineType]; dup {
		panic("parser: Register called twice for differ " + engineType)
	}
	editors[engineType] = se
}

// DeparseDatabaseEdit returns the DDL statement from DatabaseEdit structure.
func DeparseDatabaseEdit(engineType parser.EngineType, databaseEdit *api.DatabaseEdit) (string, error) {
	editorMu.RLock()
	se, ok := editors[engineType]
	editorMu.RUnlock()
	if !ok {
		return "", errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return se.DeparseDatabaseEdit(databaseEdit)
}

// ValidateDatabaseEdit validates the api message DatabaseEdit, including related column type.
func ValidateDatabaseEdit(engineType parser.EngineType, databaseEdit *api.DatabaseEdit) ([]*api.ValidateResult, error) {
	editorMu.RLock()
	se, ok := editors[engineType]
	editorMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("engine: unknown engine type %v", engineType)
	}
	return se.ValidateDatabaseEdit(databaseEdit)
}
