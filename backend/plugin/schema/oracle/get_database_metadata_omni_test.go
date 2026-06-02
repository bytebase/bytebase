package oracle

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseMetadataOmni(t *testing.T) {
	tests := []struct {
		name   string
		ddl    string
		verify func(*testing.T, *storepb.DatabaseSchemaMetadata)
	}{
		{
			name: "table_constraints_indexes_view_sequence",
			ddl: `
CREATE TABLE DEPARTMENTS (
    DEPT_ID NUMBER PRIMARY KEY,
    DEPT_NAME VARCHAR2(100) NOT NULL UNIQUE,
    MANAGER_ID NUMBER,
    CONSTRAINT chk_dept_name CHECK (DEPT_NAME IS NOT NULL)
);

CREATE TABLE EMPLOYEES (
    EMP_ID NUMBER,
    DEPT_ID NUMBER,
    EMAIL VARCHAR2(100) UNIQUE,
    SALARY NUMBER(10,2) DEFAULT 0,
    CONSTRAINT pk_employees PRIMARY KEY (EMP_ID),
    CONSTRAINT fk_emp_dept FOREIGN KEY (DEPT_ID) REFERENCES DEPARTMENTS(DEPT_ID) ON DELETE SET NULL,
    CONSTRAINT chk_salary CHECK (SALARY >= 0)
);

CREATE INDEX idx_emp_dept ON EMPLOYEES(DEPT_ID);

CREATE VIEW ACTIVE_EMPLOYEES AS
SELECT EMP_ID, DEPT_ID, EMAIL
FROM EMPLOYEES
WHERE EMAIL IS NOT NULL;

CREATE SEQUENCE emp_seq START WITH 1 INCREMENT BY 1;
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				schemaMetadata := requireSingleSchema(t, metadata)

				departments := requireTable(t, schemaMetadata, "DEPARTMENTS")
				require.Len(t, departments.Columns, 3)
				require.Equal(t, "NUMBER", departments.Columns[0].Type)
				require.False(t, departments.Columns[0].Nullable)
				require.Equal(t, "VARCHAR2(100 BYTE)", departments.Columns[1].Type)
				require.False(t, departments.Columns[1].Nullable)
				requireIndex(t, departments, "PK_DEPARTMENTS", []string{"DEPT_ID"}, true, true)
				requireIndex(t, departments, "UK_DEPARTMENTS_DEPT_NAME", []string{"DEPT_NAME"}, false, true)
				requireCheckConstraint(t, departments, "CHK_DEPT_NAME", "DEPT_NAMEISNOTNULL")

				employees := requireTable(t, schemaMetadata, "EMPLOYEES")
				require.Len(t, employees.Columns, 4)
				require.Equal(t, "0", employees.Columns[3].Default)
				requireIndex(t, employees, "PK_EMPLOYEES", []string{"EMP_ID"}, true, true)
				requireIndex(t, employees, "UK_EMPLOYEES_EMAIL", []string{"EMAIL"}, false, true)
				requireIndex(t, employees, "IDX_EMP_DEPT", []string{"DEPT_ID"}, false, false)
				requireCheckConstraint(t, employees, "CHK_SALARY", "SALARY>=0")
				requireForeignKey(t, employees, "FK_EMP_DEPT", []string{"DEPT_ID"}, "DEPARTMENTS", []string{"DEPT_ID"}, "SET NULL")

				require.Len(t, schemaMetadata.Views, 1)
				require.Equal(t, "ACTIVE_EMPLOYEES", schemaMetadata.Views[0].Name)
				require.Contains(t, schemaMetadata.Views[0].Definition, "SELECT EMP_ID, DEPT_ID, EMAIL")

				require.Len(t, schemaMetadata.Sequences, 1)
				require.Equal(t, "EMP_SEQ", schemaMetadata.Sequences[0].Name)
				require.Equal(t, "1", schemaMetadata.Sequences[0].Start)
			},
		},
		{
			name: "create_schema_wrapper",
			ddl: `
CREATE SCHEMA AUTHORIZATION HR
CREATE TABLE EMPLOYEES (
    EMP_ID NUMBER PRIMARY KEY,
    EMAIL VARCHAR2(100)
)
CREATE VIEW EMPLOYEE_VIEW AS
SELECT EMP_ID, EMAIL FROM EMPLOYEES;
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				schemaMetadata := requireSingleSchema(t, metadata)
				require.Equal(t, "HR", schemaMetadata.Name)
				employees := requireTable(t, schemaMetadata, "EMPLOYEES")
				require.Len(t, employees.Columns, 2)
				requireIndex(t, employees, "PK_EMPLOYEES", []string{"EMP_ID"}, true, true)
				view := requireView(t, schemaMetadata, "EMPLOYEE_VIEW")
				require.Contains(t, view.Definition, "SELECT EMP_ID, EMAIL FROM EMPLOYEES")
			},
		},
		{
			name: "comments_on_views_and_materialized_views",
			ddl: `
CREATE TABLE PRODUCT_SALES (
    PRODUCT_ID NUMBER NOT NULL,
    CATEGORY VARCHAR2(50) NOT NULL,
    SALES_AMOUNT NUMBER(12,2) NOT NULL
);

CREATE VIEW PRODUCT_SALES_VIEW AS
SELECT PRODUCT_ID, CATEGORY, SALES_AMOUNT
FROM PRODUCT_SALES;

CREATE MATERIALIZED VIEW PRODUCT_SALES_MV
BUILD IMMEDIATE
REFRESH COMPLETE ON DEMAND
AS
SELECT PRODUCT_ID, CATEGORY, SUM(SALES_AMOUNT) AS TOTAL_REVENUE
FROM PRODUCT_SALES
GROUP BY PRODUCT_ID, CATEGORY;

COMMENT ON VIEW PRODUCT_SALES_VIEW IS 'Product sales view';
COMMENT ON MATERIALIZED VIEW PRODUCT_SALES_MV IS 'Product sales materialized view';
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				schemaMetadata := requireSingleSchema(t, metadata)
				require.Equal(t, "Product sales view", requireView(t, schemaMetadata, "PRODUCT_SALES_VIEW").Comment)
				require.Len(t, schemaMetadata.MaterializedViews, 1)
				require.Equal(t, "PRODUCT_SALES_MV", schemaMetadata.MaterializedViews[0].Name)
				require.Equal(t, "Product sales materialized view", schemaMetadata.MaterializedViews[0].Comment)
			},
		},
		{
			name: "index_visibility_and_function_based_bitmap",
			ddl: `
CREATE TABLE ORDERS (
    ID NUMBER PRIMARY KEY,
    STATUS VARCHAR2(20)
);

CREATE INDEX idx_orders_status_invisible ON ORDERS(STATUS) INVISIBLE;
CREATE BITMAP INDEX idx_orders_lower_status ON ORDERS(LOWER(STATUS));
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				orders := requireTable(t, requireSingleSchema(t, metadata), "ORDERS")
				invisibleIndex := requireIndexMetadata(t, orders, "IDX_ORDERS_STATUS_INVISIBLE")
				require.False(t, invisibleIndex.Visible)
				functionBasedBitmapIndex := requireIndexMetadata(t, orders, "IDX_ORDERS_LOWER_STATUS")
				require.Equal(t, "FUNCTION-BASED BITMAP", functionBasedBitmapIndex.Type)
				require.True(t, functionBasedBitmapIndex.Visible)
			},
		},
		{
			name: "instead_of_trigger_on_view",
			ddl: `
CREATE TABLE EMPLOYEES (
    EMP_ID NUMBER PRIMARY KEY,
    EMAIL VARCHAR2(100)
);

CREATE VIEW EMPLOYEE_VIEW AS
SELECT EMP_ID, EMAIL
FROM EMPLOYEES;

CREATE OR REPLACE TRIGGER employee_view_insert_trg
INSTEAD OF INSERT ON EMPLOYEE_VIEW
FOR EACH ROW
BEGIN
    NULL;
END;
/
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				schemaMetadata := requireSingleSchema(t, metadata)
				view := requireView(t, schemaMetadata, "EMPLOYEE_VIEW")
				require.Len(t, view.Triggers, 1)
				require.Equal(t, "EMPLOYEE_VIEW_INSERT_TRG", view.Triggers[0].Name)
				require.Nil(t, findTable(schemaMetadata, "EMPLOYEE_VIEW"))
			},
		},
		{
			name: "materialized_view_with_index",
			ddl: `
CREATE TABLE PRODUCT_SALES (
    PRODUCT_ID NUMBER NOT NULL,
    CATEGORY VARCHAR2(50) NOT NULL,
    SALES_AMOUNT NUMBER(12,2) NOT NULL
);

CREATE MATERIALIZED VIEW PRODUCT_SALES_MV
BUILD IMMEDIATE
REFRESH COMPLETE ON DEMAND
AS
SELECT PRODUCT_ID, CATEGORY, SUM(SALES_AMOUNT) AS TOTAL_REVENUE
FROM PRODUCT_SALES
GROUP BY PRODUCT_ID, CATEGORY;

CREATE INDEX idx_mv_category ON PRODUCT_SALES_MV(CATEGORY);
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				schemaMetadata := requireSingleSchema(t, metadata)
				requireTable(t, schemaMetadata, "PRODUCT_SALES")
				require.Len(t, schemaMetadata.MaterializedViews, 1)
				require.Equal(t, "PRODUCT_SALES_MV", schemaMetadata.MaterializedViews[0].Name)
				require.Contains(t, schemaMetadata.MaterializedViews[0].Definition, "SUM(SALES_AMOUNT)")
				requireMaterializedViewIndex(t, schemaMetadata.MaterializedViews[0], "IDX_MV_CATEGORY", []string{"CATEGORY"}, false, false)
				require.Nil(t, findTable(schemaMetadata, "PRODUCT_SALES_MV"))
			},
		},
		{
			name: "alter_table_add_foreign_key",
			ddl: `
CREATE TABLE DEPARTMENTS (
    DEPT_ID NUMBER PRIMARY KEY,
    MANAGER_ID NUMBER
);

CREATE TABLE EMPLOYEES (
    EMP_ID NUMBER PRIMARY KEY
);

ALTER TABLE DEPARTMENTS ADD CONSTRAINT fk_dept_manager
    FOREIGN KEY (MANAGER_ID) REFERENCES EMPLOYEES(EMP_ID) ON DELETE SET NULL;
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				schemaMetadata := requireSingleSchema(t, metadata)
				departments := requireTable(t, schemaMetadata, "DEPARTMENTS")
				requireForeignKey(t, departments, "FK_DEPT_MANAGER", []string{"MANAGER_ID"}, "EMPLOYEES", []string{"EMP_ID"}, "SET NULL")
			},
		},
		{
			name: "alter_table_add_column",
			ddl: `
CREATE TABLE EMPLOYEES (
    EMP_ID NUMBER PRIMARY KEY
);

ALTER TABLE EMPLOYEES ADD (
    EMAIL VARCHAR2(100) NOT NULL,
    STATUS VARCHAR2(20) DEFAULT 'ACTIVE'
);
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				employees := requireTable(t, requireSingleSchema(t, metadata), "EMPLOYEES")
				require.Len(t, employees.Columns, 3)
				require.Equal(t, "EMAIL", employees.Columns[1].Name)
				require.Equal(t, "VARCHAR2(100 BYTE)", employees.Columns[1].Type)
				require.False(t, employees.Columns[1].Nullable)
				require.Equal(t, "STATUS", employees.Columns[2].Name)
				require.Equal(t, "VARCHAR2(20 BYTE)", employees.Columns[2].Type)
				require.Equal(t, "'ACTIVE'", employees.Columns[2].Default)
			},
		},
		{
			name: "inline_constraints_and_omitted_reference_columns",
			ddl: `
CREATE TABLE DEPARTMENTS (
    DEPT_ID NUMBER PRIMARY KEY
);

CREATE TABLE EMPLOYEES (
    EMP_ID NUMBER PRIMARY KEY,
    SALARY NUMBER CHECK (SALARY > 0) CHECK (SALARY < 1000000),
    DEPT_ID NUMBER REFERENCES DEPARTMENTS,
    MANAGER_DEPT_ID NUMBER,
    CONSTRAINT fk_manager_dept FOREIGN KEY (MANAGER_DEPT_ID) REFERENCES DEPARTMENTS
);
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				employees := requireTable(t, requireSingleSchema(t, metadata), "EMPLOYEES")
				requireCheckConstraint(t, employees, "CHK_EMPLOYEES_SALARY", "SALARY>0")
				requireCheckConstraint(t, employees, "CHK_EMPLOYEES_SALARY_2", "SALARY<1000000")
				requireForeignKey(t, employees, "FK_EMPLOYEES_DEPT_ID", []string{"DEPT_ID"}, "DEPARTMENTS", []string{"DEPT_ID"}, "")
				requireForeignKey(t, employees, "FK_MANAGER_DEPT", []string{"MANAGER_DEPT_ID"}, "DEPARTMENTS", []string{"DEPT_ID"}, "")
			},
		},
		{
			name: "virtual_column",
			ddl: `
CREATE TABLE ORDER_ITEMS (
    QTY NUMBER,
    PRICE NUMBER,
    TOTAL AS (QTY * PRICE)
);
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				orderItems := requireTable(t, requireSingleSchema(t, metadata), "ORDER_ITEMS")
				require.Len(t, orderItems.Columns, 3)
				require.Equal(t, "TOTAL", orderItems.Columns[2].Name)
				require.Equal(t, "NUMBER", orderItems.Columns[2].Type)
				require.Equal(t, "QTY*PRICE", orderItems.Columns[2].Default)
			},
		},
		{
			name: "escaped_string_literals_in_expressions",
			ddl: `
CREATE TABLE AUTHORS (
    NAME VARCHAR2(100),
    IS_REILLY AS (CASE WHEN NAME = 'O''Reilly' THEN 1 ELSE 0 END),
    CONSTRAINT chk_author_name CHECK (NAME <> 'O''Reilly')
);
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				authors := requireTable(t, requireSingleSchema(t, metadata), "AUTHORS")
				require.Len(t, authors.Columns, 2)
				require.Equal(t, "CASEWHENNAME='O''Reilly'THEN1ELSE0END", authors.Columns[1].Default)
				requireCheckConstraint(t, authors, "CHK_AUTHOR_NAME", "NAME<>'O''Reilly'")
			},
		},
		{
			name: "default_on_null_column",
			ddl: `
CREATE TABLE ORDERS (
    ID NUMBER PRIMARY KEY,
    STATUS VARCHAR2(20) DEFAULT ON NULL 'PENDING'
);
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				orders := requireTable(t, requireSingleSchema(t, metadata), "ORDERS")
				require.Len(t, orders.Columns, 2)
				require.Equal(t, "STATUS", orders.Columns[1].Name)
				require.Equal(t, "'PENDING'", orders.Columns[1].Default)
				require.True(t, orders.Columns[1].DefaultOnNull)
			},
		},
		{
			name: "package_metadata",
			ddl: `
CREATE OR REPLACE PACKAGE financial_utils AS
    c_default_currency CONSTANT VARCHAR2(3) := 'USD';
    FUNCTION format_currency(p_amount NUMBER) RETURN VARCHAR2;
END financial_utils;
/

CREATE OR REPLACE PACKAGE BODY financial_utils AS
    FUNCTION format_currency(p_amount NUMBER) RETURN VARCHAR2 IS
    BEGIN
        RETURN TO_CHAR(p_amount, 'FM9999990.00');
    END format_currency;
END financial_utils;
/
`,
			verify: func(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) {
				schemaMetadata := requireSingleSchema(t, metadata)
				require.Len(t, schemaMetadata.Packages, 1)
				require.Equal(t, "FINANCIAL_UTILS", schemaMetadata.Packages[0].Name)
				require.Contains(t, schemaMetadata.Packages[0].Definition, "c_default_currency")
				require.Contains(t, schemaMetadata.Packages[0].Definition, "FUNCTION format_currency")
				require.Contains(t, schemaMetadata.Packages[0].Definition, "PACKAGE BODY financial_utils")
				require.Contains(t, schemaMetadata.Packages[0].Definition, "RETURN TO_CHAR")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := GetDatabaseMetadataOmni(tc.ddl)
			require.NoError(t, err)
			tc.verify(t, metadata)
		})
	}
}

func TestGetDatabaseMetadataUsesOmni(t *testing.T) {
	metadata, err := schema.GetDatabaseMetadata(storepb.Engine_ORACLE, `
CREATE TABLE T (
    ID NUMBER PRIMARY KEY
);
`)
	require.NoError(t, err)

	table := requireTable(t, requireSingleSchema(t, metadata), "T")
	require.Len(t, table.Columns, 1)
	requireIndex(t, table, "PK_T", []string{"ID"}, true, true)
}

func TestGetDatabaseMetadataOmniParsesSlashTerminatedPLSQLScript(t *testing.T) {
	metadata, err := GetDatabaseMetadataOmni(`
CREATE TABLE AUDIT_LOG (
    LOG_ID NUMBER PRIMARY KEY
);

CREATE OR REPLACE TRIGGER audit_log_trigger
    BEFORE INSERT ON AUDIT_LOG
    FOR EACH ROW
BEGIN
    NULL;
END;
/
`)
	require.NoError(t, err)

	table := requireTable(t, requireSingleSchema(t, metadata), "AUDIT_LOG")
	require.Len(t, table.Columns, 1)
	requireIndex(t, table, "PK_AUDIT_LOG", []string{"LOG_ID"}, true, true)
}

func requireSingleSchema(t *testing.T, metadata *storepb.DatabaseSchemaMetadata) *storepb.SchemaMetadata {
	t.Helper()
	require.Len(t, metadata.Schemas, 1)
	return metadata.Schemas[0]
}

func requireTable(t *testing.T, schemaMetadata *storepb.SchemaMetadata, name string) *storepb.TableMetadata {
	t.Helper()
	table := findTable(schemaMetadata, name)
	require.NotNil(t, table)
	return table
}

func requireView(t *testing.T, schemaMetadata *storepb.SchemaMetadata, name string) *storepb.ViewMetadata {
	t.Helper()
	for _, view := range schemaMetadata.Views {
		if view.Name == name {
			return view
		}
	}
	t.Fatalf("view %q not found", name)
	return nil
}

func findTable(schemaMetadata *storepb.SchemaMetadata, name string) *storepb.TableMetadata {
	for _, table := range schemaMetadata.Tables {
		if table.Name == name {
			return table
		}
	}
	return nil
}

func requireIndex(t *testing.T, table *storepb.TableMetadata, name string, expressions []string, primary bool, unique bool) {
	t.Helper()
	index := requireIndexMetadata(t, table, name)
	require.Equal(t, expressions, index.Expressions)
	require.Equal(t, primary, index.Primary)
	require.Equal(t, unique, index.Unique)
}

func requireIndexMetadata(t *testing.T, table *storepb.TableMetadata, name string) *storepb.IndexMetadata {
	t.Helper()
	for _, index := range table.Indexes {
		if index.Name == name {
			return index
		}
	}
	t.Fatalf("index %q not found in table %q", name, table.Name)
	return nil
}

func requireMaterializedViewIndex(t *testing.T, materializedView *storepb.MaterializedViewMetadata, name string, expressions []string, primary bool, unique bool) {
	t.Helper()
	for _, index := range materializedView.Indexes {
		if index.Name != name {
			continue
		}
		require.Equal(t, expressions, index.Expressions)
		require.Equal(t, primary, index.Primary)
		require.Equal(t, unique, index.Unique)
		return
	}
	t.Fatalf("index %q not found in materialized view %q", name, materializedView.Name)
}

func requireCheckConstraint(t *testing.T, table *storepb.TableMetadata, name string, expression string) {
	t.Helper()
	for _, check := range table.CheckConstraints {
		if check.Name != name {
			continue
		}
		require.Equal(t, expression, check.Expression)
		return
	}
	t.Fatalf("check constraint %q not found in table %q", name, table.Name)
}

func requireForeignKey(t *testing.T, table *storepb.TableMetadata, name string, columns []string, referencedTable string, referencedColumns []string, onDelete string) {
	t.Helper()
	for _, fk := range table.ForeignKeys {
		if fk.Name != name {
			continue
		}
		require.Equal(t, columns, fk.Columns)
		require.Equal(t, referencedTable, fk.ReferencedTable)
		require.Equal(t, referencedColumns, fk.ReferencedColumns)
		require.Equal(t, onDelete, fk.OnDelete)
		return
	}
	t.Fatalf("foreign key %q not found in table %q", name, table.Name)
}
