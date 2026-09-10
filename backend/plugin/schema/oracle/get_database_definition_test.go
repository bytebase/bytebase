package oracle

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
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

// TestGetDatabaseDefinitionWithTestcontainer tests the GetDatabaseDefinition function
// by creating a schema, getting its definition, recreating it in a new database,
// and comparing the results.
//
//nolint:tparallel
func TestGetDatabaseDefinitionWithTestcontainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Oracle testcontainer test in short mode")
	}

	ctx := context.Background()

	container := testcontainer.SharedOracleContainer(t)

	// Get SYSTEM database connection for user management
	systemDB := container.GetDB()

	// Test cases with various schema configurations
	testCases := []struct {
		name          string
		initialSchema string
		description   string
	}{
		{
			name: "basic_tables_and_constraints",
			initialSchema: `
CREATE TABLE DEPARTMENTS (
    ID NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    NAME VARCHAR2(100) NOT NULL,
    BUDGET NUMBER(12, 2) DEFAULT 0
);

CREATE TABLE EMPLOYEES (
    ID NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    NAME VARCHAR2(100) NOT NULL,
    EMAIL VARCHAR2(100) UNIQUE,
    DEPARTMENT_ID NUMBER,
    SALARY NUMBER(10, 2),
    HIRE_DATE DATE DEFAULT SYSDATE,
    IS_ACTIVE NUMBER(1) DEFAULT 1 CHECK (IS_ACTIVE IN (0, 1)),
    CONSTRAINT FK_DEPT FOREIGN KEY (DEPARTMENT_ID) REFERENCES DEPARTMENTS(ID)
);

CREATE INDEX IDX_EMP_DEPT ON EMPLOYEES(DEPARTMENT_ID);
CREATE INDEX IDX_EMP_NAME ON EMPLOYEES(NAME);
`,
			description: "Basic tables with primary keys, foreign keys, unique constraints, check constraints, and indexes",
		},
		{
			name: "view_with_instead_of_trigger",
			initialSchema: `
CREATE TABLE EMPLOYEES (
    EMP_ID NUMBER PRIMARY KEY,
    EMAIL VARCHAR2(100)
);

CREATE VIEW EMPLOYEE_VIEW AS
SELECT EMP_ID, EMAIL
FROM EMPLOYEES;

CREATE OR REPLACE TRIGGER EMPLOYEE_VIEW_INSERT_TRG
INSTEAD OF INSERT ON EMPLOYEE_VIEW
FOR EACH ROW
BEGIN
    INSERT INTO EMPLOYEES (EMP_ID, EMAIL) VALUES (:NEW.EMP_ID, :NEW.EMAIL);
END;
/
`,
			description: "A view whose INSTEAD OF trigger must survive the definition round trip",
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create unique Oracle user for this test (Oracle users are schemas)
			// Use UUID to ensure uniqueness and avoid name collisions
			testUser := fmt.Sprintf("U_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))

			// Create user using shared SYSTEM connection
			require.NoError(t, createOracleUser(systemDB, testUser))

			// Connect as the test user
			driver, err := createOracleDriver(ctx, container.GetHost(), container.GetPort(), testUser)
			require.NoError(t, err)
			defer driver.Close(ctx)

			// Step 1: Initialize the database schema and use SyncDBSchema to get metadata A
			err = executeStatements(ctx, driver, tc.initialSchema)
			require.NoError(t, err, "Failed to execute initial schema")

			// Get schema metadata A
			metadataA, err := driver.SyncDBSchema(ctx)
			require.NoError(t, err, "Failed to get initial metadata")

			// Step 2: Call GetDatabaseDefinition to generate the database definition X
			definition, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, metadataA)
			require.NoError(t, err, "Failed to generate database definition")
			require.NotEmpty(t, definition, "Generated definition should not be empty")

			// Log the generated definition for debugging
			t.Logf("Generated definition:\n%s", definition)

			// Step 3: Create a new user and run the database definition X
			// Create a second unique user for recreation test
			testUser2 := fmt.Sprintf("U_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))

			// Create the second user
			require.NoError(t, createOracleUser(systemDB, testUser2))

			// Connect as the second test user
			driver2, err := createOracleDriver(ctx, container.GetHost(), container.GetPort(), testUser2)
			require.NoError(t, err)
			defer driver2.Close(ctx)

			// Execute the generated definition in the new user's schema
			err = executeStatements(ctx, driver2, definition)
			require.NoError(t, err, "Failed to execute generated definition")

			// Get metadata B after recreating from definition
			metadataB, err := driver2.SyncDBSchema(ctx)
			require.NoError(t, err, "Failed to get metadata after recreation")

			// Step 4: Compare the database metadata A and B, should be the same
			normalizeMetadataForComparison(metadataA)
			normalizeMetadataForComparison(metadataB)

			// Normalize column positions to 0 before comparison
			normalizeColumnPositions(metadataA)
			normalizeColumnPositions(metadataB)

			// Additional normalization for Oracle-specific issues
			normalizeOracleMetadata(metadataA)
			normalizeOracleMetadata(metadataB)

			// Use cmp with protocmp for proto message comparison
			if diff := cmp.Diff(metadataA, metadataB, protocmp.Transform()); diff != "" {
				t.Errorf("Schema mismatch after recreation (-original +recreated):\n%s", diff)
			}
		})
	}
}

// normalizeOracleMetadata handles Oracle-specific normalization for metadata comparison
func normalizeOracleMetadata(metadata *storepb.DatabaseSchemaMetadata) {
	for _, schema := range metadata.Schemas {
		// A view's dependency columns record their owning Oracle user, and this
		// test recreates the schema under a second user by construction, so the
		// two can never match.
		for _, view := range schema.Views {
			for _, dependency := range view.DependencyColumns {
				dependency.Schema = ""
			}
		}
		for _, view := range schema.MaterializedViews {
			for _, dependency := range view.DependencyColumns {
				dependency.Schema = ""
			}
		}

		for _, table := range schema.Tables {
			for _, column := range table.Columns {
				// Normalize system-generated sequence references in default expressions
				if column.Default != "" {
					// If it's a system-generated sequence, remove the default expression
					// since these sequences can't be manually recreated with the same name
					if strings.Contains(column.Default, "ISEQ$$_") {
						column.Default = ""
					}
				}

				// Clear collation information as we skip it in DDL generation
				column.Collation = ""

				// Normalize NVARCHAR2 type size differences
				// Oracle stores NVARCHAR2 sizes in max bytes, but DDL defines in characters
				if strings.HasPrefix(column.Type, "NVARCHAR2") {
					// Extract the size and convert to character count
					if strings.Contains(column.Type, "(") && strings.Contains(column.Type, ")") {
						// For simplicity, we'll normalize all NVARCHAR2 types to just NVARCHAR2
						// This avoids issues with byte vs character counting
						column.Type = "NVARCHAR2"
					}
				}
			}
		}
	}
}
