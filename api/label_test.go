package api

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/jsonapi"
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
	databasePatch := &DatabasePatch{
		ID:        1,
		UpdaterID: 1,
	}
	if err := jsonapi.UnmarshalPayload(bytes.NewReader(b), databasePatch); err != nil {
		t.Fatal(err)
	}
	var labels []*DatabaseLabel
	if err := json.Unmarshal([]byte(*databasePatch.Labels), &labels); err != nil {
		t.Fatal(err)
	}
	for i, label := range labels {
		if keys[i] != label.Key || values[i] != label.Value {
			t.Fatal("Key value pair does not match!")
		}
	}

}
