package base

import (
	"fmt"
	"sync"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	mux             sync.Mutex
	QueryValidators = make(map[storepb.Engine]ValidateSQLForEditorFunc)
	FieldMaskers    = make(map[storepb.Engine]GetMaskedFieldsFunc)
)

type ValidateSQLForEditorFunc func(string) bool

type GetMaskedFieldsFunc func(string, string, *db.SensitiveSchemaInfo) ([]db.SensitiveField, error)

func RegisterQueryValidator(engine storepb.Engine, f ValidateSQLForEditorFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := QueryValidators[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	QueryValidators[engine] = f
}

func RegisterGetMaskedFieldsFunc(engine storepb.Engine, f GetMaskedFieldsFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := FieldMaskers[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	FieldMaskers[engine] = f
}
