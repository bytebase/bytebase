package api

import "context"

type IssueSubscriber struct {
	// Domain specific fields
	IssueId      int `jsonapi:"attr,issueId"`
	SubscriberId int
	Subscriber   *Principal `jsonapi:"attr,subscriber"`
}

type IssueSubscriberCreate struct {
	// Domain specific fields
	IssueId      int
	SubscriberId int `jsonapi:"attr,subscriberId"`
}

type IssueSubscriberFind struct {
	// Domain specific fields
	IssueId      *int
	SubscriberId *int
}

type IssueSubscriberDelete struct {
	// Domain specific fields
	IssueId      int
	SubscriberId int
}

type IssueSubscriberService interface {
	CreateIssueSubscriber(ctx context.Context, create *IssueSubscriberCreate) (*IssueSubscriber, error)
	FindIssueSubscriberList(ctx context.Context, find *IssueSubscriberFind) ([]*IssueSubscriber, error)
	DeleteIssueSubscriber(ctx context.Context, delete *IssueSubscriberDelete) error
}
