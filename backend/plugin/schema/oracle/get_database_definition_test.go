package oracle

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGetTableDefinition(t *testing.T) {
	tests := []struct {
		name  string
		table *storepb.TableMetadata
		want  string
	}{
		{
			// A table whose last column has DEFAULT + NOT NULL, followed by a
			// table-level FOREIGN KEY with no other indexes or constraints.
			// The FK must be separated from the column by a comma (ORA-02253).
			name: "foreign key after default not null column without other constraints",
			table: &storepb.TableMetadata{
				Name: "T1",
				Columns: []*storepb.ColumnMetadata{
					{Name: "C1", Type: "NUMBER", Nullable: false},
					{Name: "C2", Type: "NUMBER(1)", Default: "0", Nullable: false},
				},
				ForeignKeys: []*storepb.ForeignKeyMetadata{
					{
						Name:              "T1_FK",
						Columns:           []string{"C1"},
						ReferencedTable:   "T2",
						ReferencedColumns: []string{"C1"},
					},
				},
			},
			want: `CREATE TABLE "T1" (
  "C1" NUMBER NOT NULL,
  "C2" NUMBER(1) DEFAULT 0 NOT NULL,
  CONSTRAINT "T1_FK" FOREIGN KEY ("C1") REFERENCES "T2" ("C1")
);

`,
		},
		{
			name: "check constraint after column without other constraints",
			table: &storepb.TableMetadata{
				Name: "T1",
				Columns: []*storepb.ColumnMetadata{
					{Name: "C1", Type: "NUMBER", Nullable: false},
				},
				CheckConstraints: []*storepb.CheckConstraintMetadata{
					{Name: "T1_CK", Expression: `"C1" > 0`},
				},
			},
			want: `CREATE TABLE "T1" (
  "C1" NUMBER NOT NULL,
  CONSTRAINT "T1_CK" CHECK ("C1" > 0)
);

`,
		},
		{
			name: "primary key, check, and foreign key",
			table: &storepb.TableMetadata{
				Name: "T1",
				Columns: []*storepb.ColumnMetadata{
					{Name: "C1", Type: "NUMBER", Nullable: false},
					{Name: "C2", Type: "NUMBER(1)", Default: "0", Nullable: false},
				},
				Indexes: []*storepb.IndexMetadata{
					{
						Name:         "T1_PK",
						Expressions:  []string{"C1"},
						Primary:      true,
						Unique:       true,
						IsConstraint: true,
					},
				},
				CheckConstraints: []*storepb.CheckConstraintMetadata{
					{Name: "T1_CK", Expression: `"C2" IN (0, 1)`},
				},
				ForeignKeys: []*storepb.ForeignKeyMetadata{
					{
						Name:              "T1_FK",
						Columns:           []string{"C1"},
						ReferencedTable:   "T2",
						ReferencedColumns: []string{"C1"},
					},
				},
			},
			want: `CREATE TABLE "T1" (
  "C1" NUMBER NOT NULL,
  "C2" NUMBER(1) DEFAULT 0 NOT NULL,
  CONSTRAINT "T1_PK" PRIMARY KEY ("C1"),
  CONSTRAINT "T1_CK" CHECK ("C2" IN (0, 1)),
  CONSTRAINT "T1_FK" FOREIGN KEY ("C1") REFERENCES "T2" ("C1")
);

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTableDefinition("", tt.table, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
