package api

import (
	"context"
	"encoding/json"
)

// InboxStatus is the status for inboxes.
type InboxStatus string

const (
	// Unread is the inbox status for UNREAD.
	Unread InboxStatus = "UNREAD"
	// Read is the inbox status for READ.
	Read InboxStatus = "READ"
)

func (e InboxStatus) String() string {
	switch e {
	case Unread:
		return "UNREAD"
	case Read:
		return "READ"
	}
	return "UNKNOWN"
}

// InboxRaw is the store model for an Inbox.
// Fields have exactly the same meanings as Inbox.
type InboxRaw struct {
	ID int

	// Domain specific fields
	ReceiverID  int
	ActivityRaw *ActivityRaw
	Status      InboxStatus
}

// ToInbox creates an instance of Inbox based on the InboxRaw.
// This is intended to be called when we need to compose an Inbox relationship.
func (raw *InboxRaw) ToInbox() *Inbox {
	return &Inbox{
		ID: raw.ID,

		ReceiverID: raw.ReceiverID,
		Status:     raw.Status,
	}
}

// Inbox is the API message for an inbox.
type Inbox struct {
	ID int `jsonapi:"primary,inbox"`

	// Domain specific fields
	ReceiverID int         `jsonapi:"attr,receiverId"`
	Activity   *Activity   `jsonapi:"relation,activity"`
	Status     InboxStatus `jsonapi:"attr,status"`
}

// InboxCreate is the API message for creating an inbox.
type InboxCreate struct {
	// Domain specific fields
	ReceiverID int
	ActivityID int
}

// InboxFind is the API message for finding inboxes.
type InboxFind struct {
	ID *int

	// Domain specific fields
	ReceiverID *int
	// If specified, then it will only fetch "UNREAD" item or "READ" item whose activity created after "CreatedAfterTs"
	ReadCreatedAfterTs *int64
}

func (find *InboxFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// InboxPatch is the API message for patching an inbox.
type InboxPatch struct {
	ID int

	// Domain specific fields
	Status InboxStatus `jsonapi:"attr,status"`
}

// InboxSummary is the API message for inbox summary info.
// This is used by the frontend to render the inbox sidebar item without fetching the actual inbox items.
// This returns json instead of jsonapi since it't not dealing with a particular resource.
type InboxSummary struct {
	HasUnread      bool `json:"hasUnread"`
	HasUnreadError bool `json:"hasUnreadError"`
}

// InboxService is the service for inboxes.
type InboxService interface {
	CreateInbox(ctx context.Context, create *InboxCreate) (*InboxRaw, error)
	// Find the inbox list and return most recent created item first.
	FindInboxList(ctx context.Context, find *InboxFind) ([]*InboxRaw, error)
	FindInbox(ctx context.Context, find *InboxFind) (*InboxRaw, error)
	PatchInbox(ctx context.Context, patch *InboxPatch) (*InboxRaw, error)
	FindInboxSummary(ctx context.Context, principalID int) (*InboxSummary, error)
}
