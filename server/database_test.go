package server

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
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
			name: "invalid label value",
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
					ValueList: []string{"moon"},
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

func TestRestoreDatabaseEdit(t *testing.T) {
	var (
		defaultValue = "0"
	)

	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "create table t1",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				EngineType: db.MySQL,
				CreateTableList: []*api.CreateTableContext{
					{
						Name:   "t1",
						Type:   "BASE TABLE",
						Engine: "InnoDB",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (`id` INT NOT NULL) ENGINE = InnoDB;",
		},
		{
			name: "create table t1&t2",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				EngineType: db.MySQL,
				CreateTableList: []*api.CreateTableContext{
					{
						Name:   "t1",
						Type:   "BASE TABLE",
						Engine: "InnoDB",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
						},
					},
					{
						Name:   "t2",
						Type:   "BASE TABLE",
						Engine: "InnoDB",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (`id` INT NOT NULL) ENGINE = InnoDB;CREATE TABLE `t2` (`id` INT NOT NULL) ENGINE = InnoDB;",
		},
		{
			name: "create table t1 with name",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				EngineType: db.MySQL,
				CreateTableList: []*api.CreateTableContext{
					{
						Name:   "t1",
						Type:   "BASE TABLE",
						Engine: "InnoDB",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     "id",
								Type:     "int",
								Default:  &defaultValue,
								Nullable: true,
								Comment:  "ID",
							},
							{
								Name:    "name",
								Type:    "varchar",
								Comment: "Name",
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (`id` INT COMMENT 'ID' DEFAULT '0',`name` VARCHAR CHARACTER SET UTF8MB4 COLLATE utf8mb4_bin COMMENT 'Name' NOT NULL) ENGINE = InnoDB;",
		},
	}

	for _, test := range tests {
		stmt, err := restoreDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}
