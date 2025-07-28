//nolint:revive
package common

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	// Just choose an empirical timeout value. The default 15 minutes seems too long.
	timeout         = 5 * time.Minute
	initialInterval = 5 * time.Second
)

// Retry uses exponential backoff with timeout.
func Retry(ctx context.Context, fn func() error) error {
	b := backoff.WithMaxRetries(
		backoff.NewExponentialBackOff(
			backoff.WithMaxElapsedTime(timeout),
			backoff.WithInitialInterval(initialInterval),
		),
		3,
	)
	b.Reset()
	bWithContext := backoff.WithContext(b, ctx)
	return backoff.Retry(fn, bWithContext)
}
