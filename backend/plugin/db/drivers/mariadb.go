package drivers

import (
	"context"

	odriver "github.com/bytebase/omni/driver"
	mariadbd "github.com/bytebase/omni/mariadb/driver"
)

var mariadbSpec = mysqlFamilySpec(func(ctx context.Context, cfg odriver.Config) (odriver.Driver, error) {
	return mariadbd.Open(ctx, cfg)
})
