package api

import (
	"context"
	"encoding/json"
)

type InboxStatus string

const (
	UNREAD InboxStatus = "UNREAD"
	READ   InboxStatus = "READ"
)

func (e InboxStatus) String() string {
	switch e {
	case UNREAD:
		return "UNREAD"
	case READ:
		return "READ"
	}
	return "UNKNOWN"
}

type Inbox struct {
	ID int `jsonapi:"primary,inbox"`

	// Domain specific fields
	ReceiverID int         `jsonapi:"attr,receiverID"`
	Activity   *Activity   `jsonapi:"relation,activity"`
	Status     InboxStatus `jsonapi:"attr,status"`
}

type InboxCreate struct {
	// Domain specific fields
	ReceiverID int
	ActivityID int
}

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

type InboxPatch struct {
	ID int

	// Domain specific fields
	Status InboxStatus `jsonapi:"attr,status"`
}

// Contains the inbox summary info.
// This is used by the frontend to render the inbox sidebar item without fetching the actual inbox items.
// This returns json instead of jsonapi since it't not dealing with a particular resource.
type InboxSummary struct {
	HasUnread      bool `json:"hasUnread"`
	HasUnreadError bool `json:"hasUnreadError"`
}

type InboxService interface {
	CreateInbox(ctx context.Context, create *InboxCreate) (*Inbox, error)
	// Find the inbox list and return most recent created item first.
	FindInboxList(ctx context.Context, find *InboxFind) ([]*Inbox, error)
	FindInbox(ctx context.Context, find *InboxFind) (*Inbox, error)
	PatchInbox(ctx context.Context, patch *InboxPatch) (*Inbox, error)
	FindInboxSummary(ctx context.Context, principalID int) (*InboxSummary, error)
}
