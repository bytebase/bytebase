package common

import (
	"net/http"
)

// MockRoundTripper is a helper to mock http.RoundTripper.
type MockRoundTripper struct {
	MockRoundTrip func(r *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return m.MockRoundTrip(r)
}
