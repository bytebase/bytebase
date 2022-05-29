package oauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/common"
)

func TestPost(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, "POST body", string(body))
				return &http.Response{}, nil
			},
		},
	}
	token := "token"
	_, _, err := Post(ctx, client, "", &token, strings.NewReader("POST body"), nil)
	require.NoError(t, err)
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				return &http.Response{}, nil
			},
		},
	}
	token := "token"
	_, _, _, err := Get(ctx, client, "", &token, nil)
	require.NoError(t, err)
}

func TestPut(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, "PUT body", string(body))
				return &http.Response{}, nil
			},
		},
	}
	token := "token"
	_, _, err := Put(ctx, client, "", &token, strings.NewReader("PUT body"), nil)
	require.NoError(t, err)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				return &http.Response{}, nil
			},
		},
	}
	token := "token"
	_, _, err := Delete(ctx, client, "", &token, nil)
	require.NoError(t, err)
}

func TestRetry_Exceeded(t *testing.T) {
	ctx := context.Background()
	token := "expired"

	_, _, _, err := retry(ctx, nil, &token,
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
