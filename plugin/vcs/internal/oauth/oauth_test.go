package oauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethods(t *testing.T) {
	// todo: POST, GET, PUT, DELETE
}

func TestRetry_Exceeded(t *testing.T) {
	ctx := context.Background()
	token := "expired"

	_, _, err := retry(ctx, nil, &token,
		func(_ context.Context, _ *http.Client, _ *string) error { return nil },
		func() (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader(`
{"error":"invalid_token","error_description":"Token is expired. You can either do re-authorization or token refresh."}
`)),
			}, nil
		},
	)
	wantErr := `retries exceeded for OAuth refresher with status code 400 and body ""`
	gotErr := fmt.Sprintf("%v", err)
	assert.Equal(t, wantErr, gotErr)
}
