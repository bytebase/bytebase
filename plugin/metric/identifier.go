package metric

import "context"

// Identifier is the API message for metric identifier.
type Identifier interface {
	Identify(ctx context.Context) (*Identity, error)
}
