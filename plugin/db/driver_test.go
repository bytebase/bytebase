package db

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseMigrationInfo(t *testing.T) {
	type test struct {
		fullPath string
		baseDir  string
		want     MigrationInfo
		wantErr  string
	}

	tests := []test{
		{
			fullPath: "001foo__db1",
			baseDir:  "",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Type:        "SQL",
				Description: "Create db1 migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			fullPath: "bytebase/001foo__db1",
			baseDir:  "bytebase",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Type:        "SQL",
				Description: "Create db1 migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			fullPath: "bytebase/dev/001foo__db1",
			baseDir:  "dev",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "dev",
				Type:        "SQL",
				Description: "Create db1 migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			fullPath: "bytebase/dev/001foo__db1",
			baseDir:  "",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "dev",
				Type:        "SQL",
				Description: "Create db1 migration",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			fullPath: "001foo__db1__create_t1",
			baseDir:  "",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Type:        "SQL",
				Description: "Create t1",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			fullPath: "001foo__db1__baseline",
			baseDir:  "",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Type:        "BASELINE",
				Description: "Create db1 baseline",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			fullPath: "001foo__db1__baseline__create_t1",
			baseDir:  "",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db1",
				Database:    "db1",
				Environment: "",
				Type:        "BASELINE",
				Description: "Create t1",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			fullPath: "001foo__db_shop1__baseline__create_t1",
			baseDir:  "",
			want: MigrationInfo{
				Version:     "001foo",
				Namespace:   "db_shop1",
				Database:    "db_shop1",
				Environment: "",
				Type:        "BASELINE",
				Description: "Create t1",
				Creator:     "",
			},
			wantErr: "",
		},
		{
			fullPath: "001foo_db",
			baseDir:  "",
			want: MigrationInfo{
				Version:     "",
				Namespace:   "",
				Database:    "",
				Environment: "",
				Type:        "",
				Description: "",
				Creator:     "",
			},
			wantErr: "invalid filename format",
		},
	}

	for _, tc := range tests {
		mi, err := ParseMigrationInfo(tc.fullPath, tc.baseDir)
		if err != nil {
			if tc.wantErr == "" {
				t.Errorf("fullPath=%s, baseDir=%s: expected no error, got %w", tc.fullPath, tc.baseDir, err)
			} else if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("fullPath=%s, baseDir=%s: expected error %s, got %w", tc.fullPath, tc.baseDir, tc.wantErr, err)
			}
		} else {
			if !reflect.DeepEqual(tc.want, *mi) {
				t.Errorf("fullPath=%s, baseDir=%s: expected %+v, got %+v", tc.fullPath, tc.baseDir, tc.want, *mi)
			}
		}

	}
}
