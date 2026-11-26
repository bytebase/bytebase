// Package databricks registers Databricks's parser functions, backed by omni's
// databricks packages.
package databricks

import (
	omnianalysis "github.com/bytebase/omni/databricks/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.OmniEngine(omnianalysis.Engine), storepb.Engine_DATABRICKS)
}
