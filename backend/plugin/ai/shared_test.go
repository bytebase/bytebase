package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPResponseSizeLimit(t *testing.T) {
	for _, statusCode := range []int{http.StatusOK, http.StatusBadRequest} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			body := strings.Repeat("x", 16*1024*1024+1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(statusCode)
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			response, err := doHTTPRequest(context.Background(), request, "example", func(*HTTPError) {})
			require.ErrorContains(t, err, "response exceeds maximum size")
			require.Nil(t, response)
		})
	}
}
