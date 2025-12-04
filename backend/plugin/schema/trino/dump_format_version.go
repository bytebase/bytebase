package trino

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// DumpFormatVersion tracks the schema dump output format for Trino.
// INCREMENT THIS when modifying any code that changes Dump() or
// GetDatabaseDefinition() output for this engine.
// See bytebase/CLAUDE.md "Schema Dump Format" section for details.
const DumpFormatVersion int32 = 1

func init() {
	schema.RegisterDumpFormatVersion(storepb.Engine_TRINO, DumpFormatVersion)
}
