package api

import (
	"context"
	"encoding/json"
)

type InboxStatus string

const (
	UNREAD InboxStatus = "UNREAD"
	READ   InboxStatus = "READ"
	PINNED InboxStatus = "PINNED"
)

func (e InboxStatus) String() string {
	switch e {
	case UNREAD:
		return "UNREAD"
	case READ:
		return "READ"
	case PINNED:
		return "PINNED"
	}
	return "UNKNOWN"
}

type Inbox struct {
	ID int `jsonapi:"primary,inbox"`

	// Domain specific fields
	ReceiverId int         `jsonapi:"attr,receiverId"`
	Activity   *Activity   `jsonapi:"relation,activity"`
	Status     InboxStatus `jsonapi:"attr,status"`
}

type InboxCreate struct {
	// Domain specific fields
	ReceiverId int
	ActivityId int
}

type InboxFind struct {
	ID *int

	// Domain specific fields
	ReceiverId *int
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

type InboxService interface {
	CreateInbox(ctx context.Context, create *InboxCreate) (*Inbox, error)
	FindInboxList(ctx context.Context, find *InboxFind) ([]*Inbox, error)
	FindInbox(ctx context.Context, find *InboxFind) (*Inbox, error)
	PatchInbox(ctx context.Context, patch *InboxPatch) (*Inbox, error)
}
