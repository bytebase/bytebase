package api

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/jsonapi"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	b := []byte(`{
		"data": {
			"type": "databasePatch",
			"attributes": {
				"labels": "[{\"key\":\"bb.location\",\"value\":\"earth\"}]"
			}
		}
	}`)
	keys := []string{"bb.location", "bb.tenant"}
	values := []string{"earth", "bytebase"}
	databasePatch := &DatabasePatch{}
	err := jsonapi.UnmarshalPayload(bytes.NewReader(b), databasePatch)
	require.NoError(t, err)
	var labels []*DatabaseLabel
	err = json.Unmarshal([]byte(*databasePatch.Labels), &labels)
	require.NoError(t, err)
	for i, label := range labels {
		require.Equal(t, keys[i], label.Key)
		require.Equal(t, values[i], label.Value)
	}
}
