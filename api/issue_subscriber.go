package api

import "context"

// IssueSubscriberRaw is the store model for an IssueSubscriber.
// Fields have exactly the same meanings as IssueSubscriber.
type IssueSubscriberRaw struct {
	IssueID      int
	SubscriberID int
}

// ToIssueSubscriber creates an instance of IssueSubscriber based on the IssueSubscriberRaw.
// This is intended to be called when we need to compose an IssueSubscriber relationship.
func (raw *IssueSubscriberRaw) ToIssueSubscriber() *IssueSubscriber {
	return &IssueSubscriber{
		IssueID:      raw.IssueID,
		SubscriberID: raw.SubscriberID,
	}
}

// IssueSubscriber is the API message for an issue subscriber.
type IssueSubscriber struct {
	// Domain specific fields
	IssueID      int `jsonapi:"attr,issueId"`
	SubscriberID int
	Subscriber   *Principal `jsonapi:"relation,subscriber"`
}

// IssueSubscriberCreate is the API message for creating an issue subscriber.
type IssueSubscriberCreate struct {
	// Domain specific fields
	IssueID      int
	SubscriberID int `jsonapi:"attr,subscriberId"`
}

// IssueSubscriberFind is the API message for finding issue subscribers.
type IssueSubscriberFind struct {
	// Domain specific fields
	IssueID      *int
	SubscriberID *int
}

// IssueSubscriberDelete is the API message for deleting an issue subscriber.
type IssueSubscriberDelete struct {
	// Domain specific fields
	IssueID      int
	SubscriberID int
}

// IssueSubscriberService is the service for issue subscribers.
type IssueSubscriberService interface {
	CreateIssueSubscriber(ctx context.Context, create *IssueSubscriberCreate) (*IssueSubscriberRaw, error)
	FindIssueSubscriberList(ctx context.Context, find *IssueSubscriberFind) ([]*IssueSubscriberRaw, error)
	DeleteIssueSubscriber(ctx context.Context, delete *IssueSubscriberDelete) error
}
