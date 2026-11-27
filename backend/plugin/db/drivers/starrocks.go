package drivers

import (
	"context"

	odriver "github.com/bytebase/omni/driver"
	starrocksdriver "github.com/bytebase/omni/starrocks/driver"
)

var starrocksSpec = mysqlForkSpec(func(ctx context.Context, cfg odriver.Config) (odriver.Driver, error) {
	return starrocksdriver.Open(ctx, cfg)
})
