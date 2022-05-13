package api

import "github.com/segmentio/analytics-go"

// WorkspaceIdentify is the API message for workspace.
type WorkspaceIdentify struct {
	License string
}

// GetTraits returns the identify traits
func (w *WorkspaceIdentify) GetTraits() analytics.Traits {
	return analytics.NewTraits().Set("license", w.License)
}
