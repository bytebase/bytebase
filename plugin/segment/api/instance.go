package api

import "github.com/segmentio/analytics-go"

// InstanceEvent is the API message for instance.
type InstanceEvent struct {
	Count int `property:"count"`
}

// GetType returns the event type
func (i *InstanceEvent) GetType() EventType {
	return InstanceEventType
}

// GetProperties returns the event properties
func (i *InstanceEvent) GetProperties() analytics.Properties {
	return getProperties(i)
}
