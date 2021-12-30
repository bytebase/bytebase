package server

import (
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/kr/pretty"
)

func TestGetPeerTenantDatabase(t *testing.T) {
	dbs := []*api.Database{
		{
			ID:       0,
			Name:     "hello",
			Instance: &api.Instance{EnvironmentID: 0},
		},
		{
			ID:       1,
			Name:     "hello2",
			Instance: &api.Instance{EnvironmentID: 0},
		},
		{
			ID:       2,
			Name:     "hello",
			Instance: &api.Instance{EnvironmentID: 1},
		},
		{
			ID:       3,
			Name:     "world",
			Instance: &api.Instance{EnvironmentID: 2},
		},
	}

	tests := []struct {
		name          string
		pipeline      [][]*api.Database
		environmentID int
		want          *api.Database
	}{
		{
			"same environment",
			[][]*api.Database{
				{},
				{dbs[0], dbs[1]},
				nil,
				{dbs[3]},
				{dbs[2]},
			},
			1,
			dbs[2],
		},
		{
			"fallback",
			[][]*api.Database{
				{},
				{dbs[0], dbs[1]},
				nil,
				{dbs[3]},
			},
			1,
			dbs[0],
		},
	}

	for _, test := range tests {
		got := getPeerTenantDatabase(test.pipeline, test.environmentID)

		diff := pretty.Diff(got, test.want)
		if len(diff) > 0 {
			t.Errorf("%q: getPeerTenantDatabase() got got %+v, want %+v, diff %+v.", test.name, got, test.want, diff)
		}
	}
}
