package standard

import (
	chanalysis "github.com/bytebase/omni/clickhouse/analysis"
	hiveanalysis "github.com/bytebase/omni/hive/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_CLICKHOUSE, chanalysis.ValidateQuery)
	base.RegisterQueryValidator(storepb.Engine_HIVE, hiveanalysis.ValidateQuery)
}
