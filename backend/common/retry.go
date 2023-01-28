package common

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	// Just choose an empirical timeout value. The default 15 miniutes seems too long.
	timeout = 5 * time.Minute
)

// Retry uses exponential backoff with timeout.
func Retry(fn func() error) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = timeout
	return backoff.Retry(fn, b)
}
