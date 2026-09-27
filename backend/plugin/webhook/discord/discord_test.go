package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

func TestPostNoContentResponse(t *testing.T) {
	a := require.New(t)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		a.Equal(http.MethodPost, r.Method)
		a.Equal("application/json", r.Header.Get("Content-Type"))

		var payload Webhook
		a.NoError(json.NewDecoder(r.Body).Decode(&payload))
		a.Len(payload.EmbedList, 1)
		a.Equal(":white_check_mark: Issue approved", payload.EmbedList[0].Title)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := (&Receiver{}).Post(webhook.Context{
		URL:   server.URL,
		Level: webhook.WebhookSuccess,
		Title: "Issue approved",
	})

	a.NoError(err)
	a.Equal(1, requestCount)
}
