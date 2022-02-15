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
			filePath:         "db1__001foo",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create db1 schema migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db foo_+-=/_#?!$.__1.2.3",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}",
			want: MigrationInfo{
				Version:     "1.2.3",
				Namespace:   "db foo_+-=/_#?!$.",
				Database:    "db foo_+-=/_#?!$.",
				Environment: "",
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create db foo_+-=/_#?!$. schema migration",
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
				Engine:      VCS,
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
				Engine:      VCS,
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
				Engine:      VCS,
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
				Engine:      VCS,
				Type:        Migrate,
				Description: "Create db1 schema migration",
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
				Engine:      VCS,
				Type:        Baseline,
				Description: "Create db1 baseline",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			filePath:         "db1__001foo__baseline__create_t1",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}",
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
			filePath:         "db_shop1__001foo__baseline__create_t1",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}",
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
			filePath:         "db_shop1__001foo__data",
			filePathTemplate: "{{DB_NAME}}__{{VERSION}}__{{TYPE}}",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db_shop1",
				Database:    "db_shop1",
				Environment: "",
				Engine:      VCS,
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
				Engine:      VCS,
				Type:        Data,
				Description: "Fix customer info",
				Creator:     "",
			},
			wantErr: "",
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
	}

	for _, tc := range tests {
		mi, err := ParseMigrationInfo(tc.filePath, tc.filePathTemplate)
		if err != nil {
			if tc.wantErr == "" {
				t.Errorf("filePath=%s, filePathTemplate=%s: expected no error, got %v", tc.filePath, tc.filePathTemplate, err)
			} else if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("filePath=%s, filePathTemplate=%s: expected error %s, got %v", tc.filePath, tc.filePathTemplate, tc.wantErr, err)
			}
		} else {
			if !reflect.DeepEqual(tc.want, *mi) {
				t.Errorf("filePath=%s, filePathTemplate=%s: expected %+v, got %+v", tc.filePath, tc.filePathTemplate, tc.want, *mi)
			}
		}

	}
}
