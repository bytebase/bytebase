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
// serves. The metadata is shaped the way Oracle reports it: view and
// materialized-view definitions are a bare SELECT, and routine definitions
// start at FUNCTION/PROCEDURE with no CREATE of their own.
func TestGetObjectDefinition(t *testing.T) {
	tests := []struct {
		name string
		get  func() (string, error)
		want string
	}{
		{
			name: "view",
			get: func() (string, error) {
				return GetViewDefinition("", &storepb.ViewMetadata{
					Name:       "DEPT_EMPLOYEE_COUNT",
					Definition: "SELECT D.ID AS DEPT_ID, COUNT(E.ID) AS EMP_COUNT\nFROM DEPARTMENTS D",
				})
			},
			want: `CREATE VIEW "DEPT_EMPLOYEE_COUNT" AS SELECT D.ID AS DEPT_ID, COUNT(E.ID) AS EMP_COUNT
FROM DEPARTMENTS D;

`,
		},
		{
			name: "materialized view",
			get: func() (string, error) {
				return GetMaterializedViewDefinition("", &storepb.MaterializedViewMetadata{
					Name:       "PRODUCT_STATS",
					Definition: "SELECT PRODUCT_ID, COUNT(*) AS ORDER_COUNT FROM ORDERS GROUP BY PRODUCT_ID",
				})
			},
			want: `CREATE MATERIALIZED VIEW "PRODUCT_STATS" AS SELECT PRODUCT_ID, COUNT(*) AS ORDER_COUNT FROM ORDERS GROUP BY PRODUCT_ID;

`,
		},
		{
			// ALL_SOURCE hands back the body starting at FUNCTION, so the
			// CREATE OR REPLACE prefix has to be supplied here.
			name: "function gains the CREATE OR REPLACE prefix",
			get: func() (string, error) {
				return GetFunctionDefinition("", &storepb.FunctionMetadata{
					Name:       "CALCULATE_DISCOUNT",
					Definition: "FUNCTION CALCULATE_DISCOUNT(AMOUNT NUMBER)\nRETURN NUMBER\nIS\nBEGIN\n    RETURN AMOUNT * 0.1;\nEND;",
				})
			},
			want: `CREATE OR REPLACE FUNCTION CALCULATE_DISCOUNT(AMOUNT NUMBER)
RETURN NUMBER
IS
BEGIN
    RETURN AMOUNT * 0.1;
END;

`,
		},
		{
			name: "procedure gains the CREATE OR REPLACE prefix",
			get: func() (string, error) {
				return GetProcedureDefinition("", &storepb.ProcedureMetadata{
					Name:       "LOG_AUDIT",
					Definition: "PROCEDURE LOG_AUDIT(P_TABLE_NAME VARCHAR2)\nIS\nBEGIN\n    NULL;\nEND;",
				})
			},
			want: `CREATE OR REPLACE PROCEDURE LOG_AUDIT(P_TABLE_NAME VARCHAR2)
IS
BEGIN
    NULL;
END;

`,
		},
		{
			// ALL_VIEWS.TEXT carries only the query, so the aliases from
			// CREATE VIEW V (RENAMED_ID, ...) survive solely in view.Columns.
			name: "view keeps its explicit column aliases",
			get: func() (string, error) {
				return GetViewDefinition("", &storepb.ViewMetadata{
					Name:       "ALIASED_VIEW",
					Definition: "SELECT BASE_ID, BASE_LABEL\nFROM SOURCE_DATA",
					Columns: []*storepb.ColumnMetadata{
						{Name: "RENAMED_ID", Type: "NUMBER"},
						{Name: "RENAMED_LABEL", Type: "VARCHAR2(50 BYTE)"},
					},
				})
			},
			want: `CREATE VIEW "ALIASED_VIEW" ("RENAMED_ID", "RENAMED_LABEL") AS SELECT BASE_ID, BASE_LABEL
FROM SOURCE_DATA;

`,
		},
		{
			// An INSTEAD OF trigger carries the view's DML behavior. Dropping it
			// would hand back schema text that recreates the view read-only.
			// The body shape is constructTriggerBody's in the Oracle sync: it
			// prepends CREATE OR REPLACE TRIGGER to ALL_TRIGGERS.DESCRIPTION.
			name: "view keeps its INSTEAD OF trigger",
			get: func() (string, error) {
				return GetViewDefinition("", &storepb.ViewMetadata{
					Name:       "EMPLOYEE_VIEW",
					Definition: "SELECT EMP_ID, EMAIL FROM EMPLOYEES",
					Triggers: []*storepb.TriggerMetadata{
						{
							Name: "EMPLOYEE_VIEW_INSERT_TRG",
							Body: "CREATE OR REPLACE TRIGGER EMPLOYEE_VIEW_INSERT_TRG\nINSTEAD OF INSERT ON EMPLOYEE_VIEW\nFOR EACH ROW\nBEGIN\n    NULL;\nEND;",
						},
					},
				})
			},
			want: `CREATE VIEW "EMPLOYEE_VIEW" AS SELECT EMP_ID, EMAIL FROM EMPLOYEES;

CREATE OR REPLACE TRIGGER EMPLOYEE_VIEW_INSERT_TRG
INSTEAD OF INSERT ON EMPLOYEE_VIEW
FOR EACH ROW
BEGIN
    NULL;
END;

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestObjectDefinitionsRegisteredForOracle goes through the schema registry
// rather than calling the functions directly, because the registry lookup is
// what was missing: GetSchemaString served only DATABASE and TABLE for Oracle
// and returned "engine ORACLE is not supported" for the rest, while
// supportGetStringSchema on the frontend lists ORACLE and offers "view schema
// text" on views.
func TestObjectDefinitionsRegisteredForOracle(t *testing.T) {
	tests := []struct {
		name string
		get  func() (string, error)
	}{
		{
			name: "view",
			get: func() (string, error) {
				return schema.GetViewDefinition(storepb.Engine_ORACLE, "", &storepb.ViewMetadata{Name: "V", Definition: "SELECT 1 FROM DUAL"})
			},
		},
		{
			name: "materialized view",
			get: func() (string, error) {
				return schema.GetMaterializedViewDefinition(storepb.Engine_ORACLE, "", &storepb.MaterializedViewMetadata{Name: "MV", Definition: "SELECT 1 FROM DUAL"})
			},
		},
		{
			name: "function",
			get: func() (string, error) {
				return schema.GetFunctionDefinition(storepb.Engine_ORACLE, "", &storepb.FunctionMetadata{Name: "F", Definition: "FUNCTION F RETURN NUMBER IS BEGIN RETURN 1; END;"})
			},
		},
		{
			name: "procedure",
			get: func() (string, error) {
				return schema.GetProcedureDefinition(storepb.Engine_ORACLE, "", &storepb.ProcedureMetadata{Name: "P", Definition: "PROCEDURE P IS BEGIN NULL; END;"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get()
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}
