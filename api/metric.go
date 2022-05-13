package api

import "context"

// Workspace is the instance for console application.
type Workspace struct {
	Plan string
}

// MetricService is the service for metrics.
type MetricService interface {
	Close()
	Report(ctx context.Context)
	Identify(workspace *Workspace)
}
