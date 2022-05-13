package api

import "github.com/segmentio/analytics-go"

// Identify is the API message for Identify.
type Identify interface {
	GetTraits() analytics.Traits
}
