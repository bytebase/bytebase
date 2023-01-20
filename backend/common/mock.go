package common

import (
	"net/http"
)

// MockRoundTripper is a helper to mock http.RoundTripper.
type MockRoundTripper struct {
	MockRoundTrip func(r *http.Request) (*http.Response, error)
}

// RoundTrip is the interface for doing mock RoundTrip.
func (m *MockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return m.MockRoundTrip(r)
}
