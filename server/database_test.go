package server

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/api"
)

func TestValidateDatabaseLabelList(t *testing.T) {
	tests := []struct {
		name            string
		labelList       []*api.DatabaseLabel
		labelKeyList    []*api.LabelKey
		environmentName string
		wantErr         bool
	}{
		{
			name: "valid label list",
			labelList: []*api.DatabaseLabel{
				{
					Key:   "bb.location",
					Value: "earth",
				},
				{
					Key:   api.EnvironmentKeyName,
					Value: "Dev",
				},
			},
			labelKeyList: []*api.LabelKey{
				{
					Key:       "bb.location",
					ValueList: []string{"earth"},
				},
				{
					Key:       api.EnvironmentKeyName,
					ValueList: []string{},
				},
			},
			environmentName: "Dev",
			wantErr:         false,
		},
		{
			name: "invalid label key",
			labelList: []*api.DatabaseLabel{
				{
					Key:   "bb.location",
					Value: "earth",
				},
				{
					Key:   api.EnvironmentKeyName,
					Value: "Dev",
				},
				{
					Key:   "bb.tenant",
					Value: "bytebase",
				},
			},
			labelKeyList: []*api.LabelKey{
				{
					Key:       "bb.location",
					ValueList: []string{"earth"},
				},
				{
					Key:       api.EnvironmentKeyName,
					ValueList: []string{},
				},
			},
			environmentName: "Dev",
			wantErr:         true,
		},
		{
			name: "environment label not present",
			labelList: []*api.DatabaseLabel{
				{
					Key:   "bb.location",
					Value: "earth",
				},
			},
			labelKeyList: []*api.LabelKey{
				{
					Key:       "bb.location",
					ValueList: []string{"earth"},
				},
				{
					Key:       api.EnvironmentKeyName,
					ValueList: []string{},
				},
			},
			environmentName: "Dev",
			wantErr:         true,
		},
		{
			name: "cannot mutate environment label",
			labelList: []*api.DatabaseLabel{
				{
					Key:   "bb.location",
					Value: "earth",
				},
				{
					Key:   api.EnvironmentKeyName,
					Value: "Prod",
				},
			},
			labelKeyList: []*api.LabelKey{
				{
					Key:       "bb.location",
					ValueList: []string{"earth"},
				},
				{
					Key:       api.EnvironmentKeyName,
					ValueList: []string{},
				},
			},
			environmentName: "Dev",
			wantErr:         true,
		},
	}

	for _, test := range tests {
		err := validateDatabaseLabelList(test.labelList, test.labelKeyList, test.environmentName)
		if test.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
