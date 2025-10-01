//nolint:revive
package common

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v5"
)

const (
	// Just choose an empirical timeout value. The default 15 minutes seems too long.
	timeout         = 5 * time.Minute
	initialInterval = 5 * time.Second
)

// Retry uses exponential backoff with timeout.
func Retry(ctx context.Context, fn func() error) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initialInterval

	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	}, backoff.WithBackOff(b), backoff.WithMaxElapsedTime(timeout), backoff.WithMaxTries(3))
	return err
}
