// Package clickhouse registers ClickHouse's parser functions, each backed by
// omni's ClickHouse packages.
package clickhouse

import (
	chanalysis "github.com/bytebase/omni/clickhouse/analysis"
	chcompletion "github.com/bytebase/omni/clickhouse/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.OmniEngine(chanalysis.Engine, chcompletion.Engine), storepb.Engine_CLICKHOUSE)
}
