package segment

import "github.com/bytebase/bytebase/backend/plugin/metric"

// mockreporter is the metrics collector https://segment.com/.
type mockreporter struct {
}

// NewMockReporter creates a new instance of segment mock reporter.
func NewMockReporter() metric.Reporter {
	return &mockreporter{}
}

// Close will close the segment client.
func (*mockreporter) Close() {
}

// Report will exec all the segment reporter.
func (*mockreporter) Report(_ string, _ *metric.Metric) error {
	return nil
}

// Identify will identify the workspace with license.
func (*mockreporter) Identify(_ *metric.Identifier) error {
	return nil
}
