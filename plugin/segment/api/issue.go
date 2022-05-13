package api

import "github.com/segmentio/analytics-go"

// IssueEvent is the API message for issue.
type IssueEvent struct {
	Count int    `property:"count"`
	Type  string `property:"type"`
}

// GetType returns the event type
func (i *IssueEvent) GetType() EventType {
	return IssueEventType
}

// GetProperties returns the event properties
func (i *IssueEvent) GetProperties() analytics.Properties {
	return getProperties(i)
}
