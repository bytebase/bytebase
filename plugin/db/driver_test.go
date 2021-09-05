package db

import (
	"reflect"
	"strings"
	"testing"
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
			filePath:         "001foo__db1",
			filePathTemplate: "{{VERSION}}__{{DB_NAME}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create db1 migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "1.2.3__db foo_+-=/_#?!$.",
			filePathTemplate: "{{VERSION}}__{{DB_NAME}}",
			want: MigrationInfo{
				Version:     "1.2.3",
				Namespace:   "db foo_+-=/_#?!$.",
				Database:    "db foo_+-=/_#?!$.",
				Environment: "",
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create db foo_+-=/_#?!$. migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "bytebase/001foo__db1",
			filePathTemplate: "bytebase/{{VERSION}}__{{DB_NAME}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create db1 migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "bytebase/dev/001foo__db1",
			filePathTemplate: "bytebase/{{ENV_NAME}}/{{VERSION}}__{{DB_NAME}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "dev",
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create db1 migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "001foo__db1__create_t1",
			filePathTemplate: "{{VERSION}}__{{DB_NAME}}__{{DESCRIPTION}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create t1",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "001foo__db1__migrate",
			filePathTemplate: "{{VERSION}}__{{DB_NAME}}__{{TYPE}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create db1 migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "001foo__db1__baseline",
			filePathTemplate: "{{VERSION}}__{{DB_NAME}}__{{TYPE}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Engine:      VCS,
				Type:        Baseline,
				Description: "Create db1 baseline",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "001foo__db1__baseline__create_t1",
			filePathTemplate: "{{VERSION}}__{{DB_NAME}}__{{TYPE}}__{{DESCRIPTION}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Engine:      VCS,
				Type:        Baseline,
				Description: "Create t1",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "001foo__db_shop1__baseline__create_t1",
			filePathTemplate: "{{VERSION}}__{{DB_NAME}}__{{TYPE}}__{{DESCRIPTION}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db_shop1",
				Database:    "db_shop1",
				Environment: "",
				Engine:      VCS,
				Type:        Baseline,
				Description: "Create t1",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db",
			filePathTemplate: "{{VERSION}}__{{DB_NAME}}",
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
		mi, err := ParseMigrationInfo(tc.filePath, tc.filePathTemplate)
		if err != nil {
			if tc.wantErr == "" {
				t.Errorf("filePath=%s, filePathTemplate=%s: expected no error, got %w", tc.filePath, tc.filePathTemplate, err)
			} else if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("filePath=%s, filePathTemplate=%s: expected error %s, got %w", tc.filePath, tc.filePathTemplate, tc.wantErr, err)
			}
		} else {
			if !reflect.DeepEqual(tc.want, *mi) {
				t.Errorf("filePath=%s, filePathTemplate=%s: expected %+v, got %+v", tc.filePath, tc.filePathTemplate, tc.want, *mi)
			}
		}

	}
}
