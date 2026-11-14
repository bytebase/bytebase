package snowflake

import (
	"github.com/bytebase/omni/snowflake/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_SNOWFLAKE, analysis.ValidateQuery)
}
