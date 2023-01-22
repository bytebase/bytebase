package taskrun

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/store"
)

func TestGetPeerTenantDatabase(t *testing.T) {
	dbs := []*store.DatabaseMessage{
		{
			UID:           0,
			DatabaseName:  "hello",
			EnvironmentID: "dev",
		},
		{
			UID:           1,
			DatabaseName:  "hello2",
			EnvironmentID: "dev",
		},
		{
			UID:           2,
			DatabaseName:  "hello",
			EnvironmentID: "staging",
		},
		{
			UID:           3,
			DatabaseName:  "world",
			EnvironmentID: "prod",
		},
	}

	tests := []struct {
		name          string
		pipeline      [][]*store.DatabaseMessage
		environmentID string
		want          *store.DatabaseMessage
	}{
		{
			"same environment",
			[][]*store.DatabaseMessage{
				{},
				{dbs[0], dbs[1]},
				nil,
				{dbs[3]},
				{dbs[2]},
			},
			"staging",
			dbs[2],
		},
		{
			"fallback",
			[][]*store.DatabaseMessage{
				{},
				{dbs[0], dbs[1]},
				nil,
				{dbs[3]},
			},
			"staging",
			dbs[0],
		},
	}

	for _, test := range tests {
		got := getPeerTenantDatabase(test.pipeline, test.environmentID)
		assert.Equal(t, got, test.want)
	}
}
