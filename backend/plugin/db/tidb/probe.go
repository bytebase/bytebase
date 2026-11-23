package tidb

import (
	"context"
	"fmt"
	"time"

	omnidriver "github.com/bytebase/omni/driver"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Prober       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
)

// Probe runs a live review rule's probe. A BATCH ... DRY RUN statement passes
// Autocommit, because TiDB refuses it inside a transaction.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return d.ProbeOmni(ctx, pCtx, statement, util.OmniProbeOptions{
		BeginStatement:           "START TRANSACTION",
		RollbackStatement:        "ROLLBACK",
		RollbackWithoutCancel:    true,
		SkipEmptyPreExecutions:   true,
		CheckPreExecutionResults: true,
		FormatCell:               probeCell,
	})
}

// probeCell maps one contract value onto the Go value a probe result carries.
func probeCell(value omnidriver.Value) any {
	switch value.Kind {
	case omnidriver.Timestamp, omnidriver.TimestampTZ:
		return timestampText(value)
	default:
		return util.DefaultProbeCell(value)
	}
}

// timestampText renders a timestamp the way the session spells it, so a probe's
// cell reads the same as it did when the driver scanned the server's text.
func timestampText(value omnidriver.Value) string {
	text := value.Time.Format(time.DateTime)
	if value.Precision > 0 {
		fraction := fmt.Sprintf("%09d", value.Time.Nanosecond())
		if int(value.Precision) < len(fraction) {
			fraction = fraction[:value.Precision]
		}
		text += "." + fraction
	}
	return text
}
