// Package trino provides a SQL parser adapter for Trino, backed by the omni
// hand-written Trino parser (github.com/bytebase/omni/trino/*).
package trino

import (
	"github.com/bytebase/omni/trino/analysis"
	"github.com/bytebase/omni/trino/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.OmniEngine(analysis.Engine, completion.Engine), storepb.Engine_TRINO)
}
