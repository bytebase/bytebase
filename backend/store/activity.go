package store

import (
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// ActivityMessage is the API message for activity.
type ActivityMessage struct {
	UID       int
	CreatedTs int64
	UpdatedTs int64

	// Related fields
	CreatorUID        int
	UpdaterUID        int
	ResourceContainer string
	// The object where this activity belongs
	// e.g if Type is "bb.issue.xxx", then this field refers to the corresponding issue's id.
	ContainerUID int

	// Domain specific fields
	Type    api.ActivityType
	Level   api.ActivityLevel
	Comment string
	Payload string
}
