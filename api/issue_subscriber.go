package api

import "context"

type IssueSubscriber struct {
	// Domain specific fields
	IssueID      int `jsonapi:"attr,issueId"`
	SubscriberID int
	Subscriber   *Principal `jsonapi:"attr,subscriber"`
}

type IssueSubscriberCreate struct {
	// Domain specific fields
	IssueID      int
	SubscriberID int `jsonapi:"attr,subscriberId"`
}

type IssueSubscriberFind struct {
	// Domain specific fields
	IssueID      *int
	SubscriberID *int
}

type IssueSubscriberDelete struct {
	// Domain specific fields
	IssueID      int
	SubscriberID int
}

type IssueSubscriberService interface {
	CreateIssueSubscriber(ctx context.Context, create *IssueSubscriberCreate) (*IssueSubscriber, error)
	FindIssueSubscriberList(ctx context.Context, find *IssueSubscriberFind) ([]*IssueSubscriber, error)
	DeleteIssueSubscriber(ctx context.Context, delete *IssueSubscriberDelete) error
}
