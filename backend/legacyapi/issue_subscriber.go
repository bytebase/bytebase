package api

// IssueSubscriber is the API message for an issue subscriber.
type IssueSubscriber struct {
	// Domain specific fields
	IssueID    int        `jsonapi:"attr,issueId"`
	Subscriber *Principal `jsonapi:"relation,subscriber"`
}

// IssueSubscriberCreate is the API message for creating an issue subscriber.
type IssueSubscriberCreate struct {
	SubscriberID int `jsonapi:"attr,subscriberId"`
}
