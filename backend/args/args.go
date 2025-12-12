// Package args contains build-time injected variables.
package args

// These should be set via go build -ldflags -X 'xxxx'.
var (
	Version   = "development"
	GitCommit = "unknown"
)
