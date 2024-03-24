package oauth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
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
	_, _, _, err := Post(ctx, client, "", token, strings.NewReader("POST body"))
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
	_, _, _, err := Get(ctx, client, "", token)
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
	_, _, _, err := Put(ctx, client, "", token, strings.NewReader("PUT body"))
	require.NoError(t, err)
}

func TestPatch(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, "PATCH body", string(body))
				return &http.Response{}, nil
			},
		},
	}
	token := "token"
	_, _, _, err := Patch(ctx, client, "", token, strings.NewReader("PATCH body"))
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
	_, _, _, err := Delete(ctx, client, "", token)
	require.NoError(t, err)
}
