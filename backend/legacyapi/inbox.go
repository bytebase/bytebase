package api

// InboxStatus is the status for inboxes.
type InboxStatus string

const (
	// Unread is the inbox status for UNREAD.
	Unread InboxStatus = "UNREAD"
	// Read is the inbox status for READ.
	Read InboxStatus = "READ"
)
