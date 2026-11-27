package cassandra

import (
	"github.com/bytebase/omni/cassandra/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.OmniEngine(analysis.Engine), storepb.Engine_CASSANDRA)
}
