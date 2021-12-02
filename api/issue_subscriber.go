package api

import "context"

// IssueSubscriber is the API message for an issue subscriber.
type IssueSubscriber struct {
	// Domain specific fields
	IssueID      int `jsonapi:"attr,issueId"`
	SubscriberID int
	Subscriber   *Principal `jsonapi:"attr,subscriber"`
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
	CreateIssueSubscriber(ctx context.Context, create *IssueSubscriberCreate) (*IssueSubscriber, error)
	FindIssueSubscriberList(ctx context.Context, find *IssueSubscriberFind) ([]*IssueSubscriber, error)
	DeleteIssueSubscriber(ctx context.Context, delete *IssueSubscriberDelete) error
}
