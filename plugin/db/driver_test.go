package db

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMigrationInfo(t *testing.T) {
	type test struct {
		filePath         string
		filePathTemplate string
		want             MigrationInfo
		wantErr          string
	}

	tests := []test{
		{
			filePath:         "db1__001foo",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}",
			want: MigrationInfo{
				Version:     "001foo",
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
			filePath:         "db foo_+-=_#!$.__1.2.3",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}",
			want: MigrationInfo{
				Version:     "1.2.3",
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
			filePath:         "bytebase/db1__001foo",
			filePathTemplate: "bytebase/{{DB_NAME}}__{{VERSION}}",
			want: MigrationInfo{
				Version:     "001foo",
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
			filePath:         "bytebase/dev/db1__001foo",
			filePathTemplate: "bytebase/{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}",
			want: MigrationInfo{
				Version:     "001foo",
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
			filePath:         "db1__001foo__create_t1",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}__{{DESCRIPTION}}",
			want: MigrationInfo{
				Version:     "001foo",
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
			filePath:         "db1__001foo__migrate",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}__{{TYPE}}",
			want: MigrationInfo{
				Version:     "001foo",
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
			filePath:         "db_shop1__001foo__data",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}__{{TYPE}}",
			want: MigrationInfo{
				Version:     "001foo",
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
			filePath:         "db_shop1__001foo__data__fix_customer_info",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}",
			want: MigrationInfo{
				Version:     "001foo",
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
			filePath:         ".db__20220811.sql",
			filePathTemplate: ".{{DB_NAME}}__{{VERSION}}.sql",
			want: MigrationInfo{
				Version:     "20220811",
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
			filePath:         "db1__001foo__baseline",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}__{{TYPE}}",
			want: MigrationInfo{
				Version:     "001foo",
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
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}",
			want: MigrationInfo{
				Version:     "",
				Namespace:   "",
				Database:    "",
				Environment: "",
				Type:        "",
				Description: "",
				Creator:     "",
			},
			wantErr: "does not match file path template",
		},
		{ // Make sure the "." is escaped and should only match literals not as a wildcard
			filePath:         ".db__20220811.sql",
			filePathTemplate: "A{{DB_NAME}}__{{VERSION}}Asql",
			want: MigrationInfo{
				Version:     "",
				Namespace:   "",
				Database:    "",
				Environment: "",
				Type:        "",
				Description: "",
				Creator:     "",
			},
			wantErr: "does not match file path template",
		},
	}
	for _, tc := range tests {
		t.Run(tc.filePath, func(t *testing.T) {
			mi, err := ParseMigrationInfo(tc.filePath, tc.filePathTemplate)
			if tc.wantErr != "" {
				got := fmt.Sprintf("%v", err)
				require.Contains(t, got, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, *mi)
		})
	}
}
