package db

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/store/model"
)

func TestParseMigrationInfo(t *testing.T) {
	type test struct {
		filePath              string
		filePathTemplate      string
		allowOmitDatabaseName bool
		want                  *MigrationInfo
		wantErr               string
	}

	tests := []test{
		{
			filePath:         "db1##001foo",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Source:      VCS,
				Type:        Migrate,
				Description: "Create db1 schema migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db foo_+-=_#!$.##1.2.3",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "1.2.3"},
				Namespace:   "db foo_+-=_#!$.",
				Database:    "db foo_+-=_#!$.",
				Environment: "",
				Source:      VCS,
				Type:        Migrate,
				Description: "Create db foo_+-=_#!$. schema migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "bytebase/db1##001foo",
			filePathTemplate: "bytebase/{{DB_NAME}}##{{VERSION}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Source:      VCS,
				Type:        Migrate,
				Description: "Create db1 schema migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "bytebase/dev/db1##001foo",
			filePathTemplate: "bytebase/{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db1",
				Database:    "db1",
				Environment: "dev",
				Source:      VCS,
				Type:        Migrate,
				Description: "Create db1 schema migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db1##001foo##create_t1",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}##{{DESCRIPTION}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Source:      VCS,
				Type:        Migrate,
				Description: "Create t1",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db1##001foo##migrate",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}##{{TYPE}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Source:      VCS,
				Type:        Migrate,
				Description: "Create db1 schema migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db1##001foo##ddl",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}##{{TYPE}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Source:      VCS,
				Type:        Migrate,
				Description: "Create db1 schema migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db1##001foo##dml",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}##{{TYPE}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Source:      VCS,
				Type:        Data,
				Description: "Create db1 data change",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db_shop1##001foo##data",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}##{{TYPE}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db_shop1",
				Database:    "db_shop1",
				Environment: "",
				Source:      VCS,
				Type:        Data,
				Description: "Create db_shop1 data change",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db_shop1##001foo##data##fix_customer_info",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db_shop1",
				Database:    "db_shop1",
				Environment: "",
				Source:      VCS,
				Type:        Data,
				Description: "Fix customer info",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         ".db##20220811.sql",
			filePathTemplate: ".{{DB_NAME}}##{{VERSION}}.sql",
			want: &MigrationInfo{
				Version:     model.Version{Version: "20220811"},
				Namespace:   "db",
				Database:    "db",
				Environment: "",
				Source:      VCS,
				Type:        Migrate,
				Description: "Create db schema migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db1##001foo##baseline",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}##{{TYPE}}",
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Source:      VCS,
				Type:        Baseline,
				Description: "Create db1 baseline",
				Creator:     "",
			},
			wantErr: "contains invalid migration type \"baseline\"",
		},
		{
			filePath:         "db",
			filePathTemplate: "{{DB_NAME}}##{{VERSION}}",
			want:             nil,
			wantErr:          "",
		},
		{ // Make sure the "." is escaped and should only match literals not as a wildcard
			filePath:         ".db##20220811.sql",
			filePathTemplate: "A{{DB_NAME}}##{{VERSION}}Asql",
			want:             nil,
			wantErr:          "",
		},

		{
			filePath:         "001foo##ddl",
			filePathTemplate: "{{VERSION}}##{{TYPE}}",
			want:             nil,
			wantErr:          "does not contain {{DB_NAME}}",
		},
		{
			filePath:              "001foo##ddl",
			filePathTemplate:      "{{VERSION}}##{{TYPE}}",
			allowOmitDatabaseName: true,
			want: &MigrationInfo{
				Version:     model.Version{Version: "001foo"},
				Source:      VCS,
				Type:        Migrate,
				Description: "Create  schema migration",
			},
			wantErr: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.filePath, func(t *testing.T) {
			mi, err := ParseMigrationInfo(tc.filePath, tc.filePathTemplate, tc.allowOmitDatabaseName)
			if tc.wantErr != "" {
				got := fmt.Sprintf("%v", err)
				require.Contains(t, got, tc.wantErr)
				return
			}
			require.NoError(t, err)
			if tc.want == nil {
				require.Nil(t, mi)
			} else {
				require.NotNil(t, mi)
				require.Equal(t, *tc.want, *mi)
			}
		})
	}
}

func TestParseSchemaFileInfo(t *testing.T) {
	tests := []struct {
		name               string
		baseDirectory      string
		schemaPathTemplate string
		file               string
		schemaInfo         *MigrationInfo
	}{
		{
			name:               "no schemaPathTemplate",
			baseDirectory:      "",
			schemaPathTemplate: "",
			file:               "Test/testdb##LATEST.sql",
			schemaInfo:         nil,
		},
		{
			name:               "only has DB_NAME",
			baseDirectory:      "",
			schemaPathTemplate: "{{DB_NAME}}##LATEST.sql",
			file:               "testdb##LATEST.sql",
			schemaInfo: &MigrationInfo{
				Source:   VCS,
				Type:     Migrate,
				Database: "testdb",
			},
		},
		{
			name:               "has both ENV_ID and DB_NAME",
			baseDirectory:      "",
			schemaPathTemplate: "{{ENV_ID}}/{{DB_NAME}}##LATEST.sql",
			file:               "Test/testdb##LATEST.sql",
			schemaInfo: &MigrationInfo{
				Source:      VCS,
				Type:        Migrate,
				Environment: "Test",
				Database:    "testdb",
			},
		},

		{
			name:               "baseDirectory does not match",
			baseDirectory:      "bytebase",
			schemaPathTemplate: "{{ENV_ID}}/{{DB_NAME}}##LATEST.sql",
			file:               "Test/testdb##LATEST.sql",
			schemaInfo:         nil,
		},
		{
			name:               "baseDirectory with both ENV_ID and DB_NAME",
			baseDirectory:      "bytebase",
			schemaPathTemplate: "{{ENV_ID}}/{{DB_NAME}}##LATEST.sql",
			file:               "bytebase/Test/testdb##LATEST.sql",
			schemaInfo: &MigrationInfo{
				Source:      VCS,
				Type:        Migrate,
				Environment: "Test",
				Database:    "testdb",
			},
		},
	}
	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseSchemaFileInfo(test.baseDirectory, test.schemaPathTemplate, test.file)
			require.NoError(t, err)
			assert.Equal(t, test.schemaInfo, got)
		})
	}
}
