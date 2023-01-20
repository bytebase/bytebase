package api

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
