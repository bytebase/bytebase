package api

import (
	"reflect"

	"github.com/segmentio/analytics-go"
)

// EventType is the segment track event name.
type EventType string

var (
	// WorkspaceEventType is the track event for workspace.
	WorkspaceEventType EventType = "workspace"
	// InstanceEventType is the track event for instance.
	InstanceEventType EventType = "instance"
	// IssueEventType is the track event for issue.
	IssueEventType EventType = "issue"
)

// Event is the API message for Track.
type Event interface {
	GetType() EventType
	GetProperties() analytics.Properties
}

func getEventProperties(data interface{}) analytics.Properties {
	res := analytics.NewProperties()

	s := reflect.ValueOf(data).Elem()
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)

		name := typeOfT.Field(i).Tag.Get("property")
		if name == "" {
			continue
		}

		if f.Kind() == reflect.Ptr {
			if !f.IsNil() {
				res.Set(name, f.Elem().Interface())
			}
		} else {
			if !f.IsZero() {
				res.Set(name, f.Interface())
			}
		}
	}

	return res
}
