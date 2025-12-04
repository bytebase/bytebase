package schema

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

var dumpFormatVersions = make(map[storepb.Engine]int32)

// RegisterDumpFormatVersion registers the dump format version for an engine.
// This should be called during init() in each engine's dump_format_version.go.
func RegisterDumpFormatVersion(engine storepb.Engine, version int32) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := dumpFormatVersions[engine]; dup {
		panic(fmt.Sprintf("RegisterDumpFormatVersion called twice for %s", engine))
	}
	dumpFormatVersions[engine] = version
}

// GetDumpFormatVersion returns the current dump format version for the given engine.
// Returns 0 for unsupported engines (drift detection will be skipped).
func GetDumpFormatVersion(engine storepb.Engine) int32 {
	mux.Lock()
	defer mux.Unlock()
	return dumpFormatVersions[engine]
}
