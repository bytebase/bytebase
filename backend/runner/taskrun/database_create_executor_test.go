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
		pipeline      [][]int
		environmentID string
		want          *store.DatabaseMessage
	}{
		{
			"same environment",
			[][]int{
				{},
				{dbs[0].UID, dbs[1].UID},
				nil,
				{dbs[3].UID},
				{dbs[2].UID},
			},
			"staging",
			dbs[2],
		},
		{
			"fallback",
			[][]int{
				{},
				{dbs[0].UID, dbs[1].UID},
				nil,
				{dbs[3].UID},
			},
			"staging",
			dbs[0],
		},
	}

	for _, test := range tests {
		got := getPeerTenantDatabase(test.pipeline, test.environmentID, dbs)
		assert.Equal(t, got, test.want)
	}
}
