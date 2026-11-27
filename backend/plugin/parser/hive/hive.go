// Package hive registers Hive's parser functions, backed by omni's hive packages.
package hive

import (
	omnianalysis "github.com/bytebase/omni/hive/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.OmniEngine(omnianalysis.Engine), storepb.Engine_HIVE)
}
