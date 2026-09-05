package oracle

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
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

// TestGetObjectDefinition covers the single-object definitions GetSchemaString
// serves. Metadata shapes are taken from what Oracle actually reports: view and
// materialized-view definitions are a bare SELECT, routine definitions start at
// FUNCTION/PROCEDURE with no CREATE, and sequences carry attributes rather than
// text.
func TestGetObjectDefinition(t *testing.T) {
	t.Parallel()

	t.Run("view", func(t *testing.T) {
		t.Parallel()
		got, err := GetViewDefinition("", &storepb.ViewMetadata{
			Name:       "DEPT_EMPLOYEE_COUNT",
			Definition: "SELECT D.ID AS DEPT_ID, COUNT(E.ID) AS EMP_COUNT\nFROM DEPARTMENTS D",
		})
		require.NoError(t, err)
		require.Equal(t, `CREATE VIEW "DEPT_EMPLOYEE_COUNT" AS SELECT D.ID AS DEPT_ID, COUNT(E.ID) AS EMP_COUNT
FROM DEPARTMENTS D;

`, got)
	})

	t.Run("materialized view", func(t *testing.T) {
		t.Parallel()
		got, err := GetMaterializedViewDefinition("", &storepb.MaterializedViewMetadata{
			Name:       "PRODUCT_STATS",
			Definition: "SELECT PRODUCT_ID, COUNT(*) AS ORDER_COUNT FROM ORDERS GROUP BY PRODUCT_ID",
		})
		require.NoError(t, err)
		require.Equal(t, `CREATE MATERIALIZED VIEW "PRODUCT_STATS" AS SELECT PRODUCT_ID, COUNT(*) AS ORDER_COUNT FROM ORDERS GROUP BY PRODUCT_ID;

`, got)
	})

	t.Run("function gains the CREATE OR REPLACE prefix", func(t *testing.T) {
		t.Parallel()
		got, err := GetFunctionDefinition("", &storepb.FunctionMetadata{
			Name:       "CALCULATE_DISCOUNT",
			Definition: "FUNCTION CALCULATE_DISCOUNT(AMOUNT NUMBER)\nRETURN NUMBER\nIS\nBEGIN\n    RETURN AMOUNT * 0.1;\nEND;",
		})
		require.NoError(t, err)
		require.Equal(t, `CREATE OR REPLACE FUNCTION CALCULATE_DISCOUNT(AMOUNT NUMBER)
RETURN NUMBER
IS
BEGIN
    RETURN AMOUNT * 0.1;
END;

`, got)
	})

	t.Run("procedure gains the CREATE OR REPLACE prefix", func(t *testing.T) {
		t.Parallel()
		got, err := GetProcedureDefinition("", &storepb.ProcedureMetadata{
			Name:       "LOG_AUDIT",
			Definition: "PROCEDURE LOG_AUDIT(P_TABLE_NAME VARCHAR2)\nIS\nBEGIN\n    NULL;\nEND;",
		})
		require.NoError(t, err)
		require.Equal(t, `CREATE OR REPLACE PROCEDURE LOG_AUDIT(P_TABLE_NAME VARCHAR2)
IS
BEGIN
    NULL;
END;

`, got)
	})

	t.Run("sequence", func(t *testing.T) {
		t.Parallel()
		got, err := GetSequenceDefinition("", &storepb.SequenceMetadata{
			Name:      "ORDER_SEQ",
			Start:     "100",
			Increment: "2",
		})
		require.NoError(t, err)
		require.Equal(t, "CREATE SEQUENCE \"ORDER_SEQ\" START WITH 100 INCREMENT BY 2;\n\n", got)
	})

	// Oracle creates these for identity columns. They belong to their table's
	// DDL and cannot be created on their own, so the whole-database output skips
	// them and so does this.
	t.Run("identity-column sequence is skipped", func(t *testing.T) {
		t.Parallel()
		got, err := GetSequenceDefinition("", &storepb.SequenceMetadata{Name: "ISEQ$$_73349"})
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

// TestObjectDefinitionsRegisteredForOracle goes through the schema registry
// rather than calling the functions directly, because the registry lookup is
// what was missing: GetSchemaString served only DATABASE and TABLE for Oracle
// and returned "engine ORACLE is not supported" for the rest, while
// supportGetStringSchema on the frontend lists ORACLE and offers "view schema
// text" on views.
func TestObjectDefinitionsRegisteredForOracle(t *testing.T) {
	t.Parallel()

	view := &storepb.ViewMetadata{Name: "V", Definition: "SELECT 1 FROM DUAL"}
	materializedView := &storepb.MaterializedViewMetadata{Name: "MV", Definition: "SELECT 1 FROM DUAL"}
	function := &storepb.FunctionMetadata{Name: "F", Definition: "FUNCTION F RETURN NUMBER IS BEGIN RETURN 1; END;"}
	procedure := &storepb.ProcedureMetadata{Name: "P", Definition: "PROCEDURE P IS BEGIN NULL; END;"}
	sequence := &storepb.SequenceMetadata{Name: "S", Start: "1"}

	for name, get := range map[string]func() (string, error){
		"view": func() (string, error) { return schema.GetViewDefinition(storepb.Engine_ORACLE, "", view) },
		"materializedView": func() (string, error) {
			return schema.GetMaterializedViewDefinition(storepb.Engine_ORACLE, "", materializedView)
		},
		"function":  func() (string, error) { return schema.GetFunctionDefinition(storepb.Engine_ORACLE, "", function) },
		"procedure": func() (string, error) { return schema.GetProcedureDefinition(storepb.Engine_ORACLE, "", procedure) },
		"sequence":  func() (string, error) { return schema.GetSequenceDefinition(storepb.Engine_ORACLE, "", sequence) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := get()
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}
