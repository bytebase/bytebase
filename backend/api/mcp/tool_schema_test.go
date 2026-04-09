package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// --- Fixture builders (mirror the JSON wire format of DatabaseMetadata proto) ---

func makeMetadataResponse(schemas ...map[string]any) map[string]any {
	return map[string]any{
		"name":    "instances/prod-pg/databases/employee_db/metadata",
		"schemas": schemas,
	}
}

func makeSchema(name string, tables ...map[string]any) map[string]any {
	return map[string]any{
		"name":   name,
		"tables": tables,
	}
}

func makeSchemaWithViews(name string, views []string, tables ...map[string]any) map[string]any {
	viewList := make([]map[string]any, 0, len(views))
	for _, v := range views {
		viewList = append(viewList, map[string]any{"name": v})
	}
	return map[string]any{
		"name":   name,
		"tables": tables,
		"views":  viewList,
	}
}

func makeTable(name string, rowCount int64, columns []map[string]any, indexes []map[string]any) map[string]any {
	t := map[string]any{
		"name":     name,
		"rowCount": fmt.Sprintf("%d", rowCount),
		"columns":  columns,
	}
	if len(indexes) > 0 {
		t["indexes"] = indexes
	}
	return t
}

func makeColumn(name, typ string, nullable bool) map[string]any {
	return map[string]any{
		"name":     name,
		"type":     typ,
		"nullable": nullable,
	}
}

func makeColumnWithDefault(name, typ, defaultVal, comment string) map[string]any {
	c := makeColumn(name, typ, false)
	c["default"] = defaultVal
	c["comment"] = comment
	return c
}

func makePrimaryKeyIndex(name string, columns []string) map[string]any {
	return map[string]any{
		"name":        name,
		"expressions": columns,
		"type":        "btree",
		"unique":      true,
		"primary":     true,
	}
}

func makeIndex(name string, unique bool, columns []string) map[string]any {
	return map[string]any{
		"name":        name,
		"expressions": columns,
		"type":        "btree",
		"unique":      unique,
		"primary":     false,
	}
}

// --- Request capture / metadata mock ---

// capturedRequest stores a single GetDatabaseMetadata request body for assertions.
type capturedRequest struct {
	Name   string `json:"name"`
	Filter string `json:"filter"`
	Limit  int32  `json:"limit"`
}

// metadataMock is a stateful mock that records GetDatabaseMetadata calls and
// returns a scripted response (or a response per-call for the candidate refetch case).
type metadataMock struct {
	databases     []map[string]any
	responses     []map[string]any // scripted responses; indexed by call count
	calls         int32
	capturedMu    sync.Mutex
	capturedCalls []capturedRequest
}

func newMetadataMock(databases []map[string]any, responses ...map[string]any) *metadataMock {
	return &metadataMock{databases: databases, responses: responses}
}

func (m *metadataMock) handler() http.Handler {
	listHandler := mockListDatabases(m.databases)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "DatabaseService/GetDatabaseMetadata") {
			var captured capturedRequest
			_ = json.NewDecoder(r.Body).Decode(&captured)

			m.capturedMu.Lock()
			m.capturedCalls = append(m.capturedCalls, captured)
			m.capturedMu.Unlock()

			idx := int(atomic.AddInt32(&m.calls, 1)) - 1
			var resp map[string]any
			if idx < len(m.responses) {
				resp = m.responses[idx]
			} else if len(m.responses) > 0 {
				// Reuse last response for additional calls.
				resp = m.responses[len(m.responses)-1]
			}
			if resp == nil {
				resp = map[string]any{"name": "", "schemas": []any{}}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		listHandler.ServeHTTP(w, r)
	})
}

func (m *metadataMock) lastCall() capturedRequest {
	m.capturedMu.Lock()
	defer m.capturedMu.Unlock()
	if len(m.capturedCalls) == 0 {
		return capturedRequest{}
	}
	return m.capturedCalls[len(m.capturedCalls)-1]
}

func (m *metadataMock) callAt(idx int) capturedRequest {
	m.capturedMu.Lock()
	defer m.capturedMu.Unlock()
	if idx >= len(m.capturedCalls) {
		return capturedRequest{}
	}
	return m.capturedCalls[idx]
}

func (m *metadataMock) callCount() int {
	return int(atomic.LoadInt32(&m.calls))
}

// mockMetadataServer returns a ready-to-use mock server that handles both
// ListDatabases and GetDatabaseMetadata calls.
func mockMetadataServer(t *testing.T, databases []map[string]any, responses ...map[string]any) (*Server, *metadataMock) {
	t.Helper()
	mock := newMetadataMock(databases, responses...)
	return newTestServerWithMock(t, mock.handler()), mock
}

// employeeDB returns a standard single-database fixture (POSTGRES, employee_db).
func employeeDB() []map[string]any {
	return []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
	}
}

// --- Tests ---

func TestGetSchema_Summary(t *testing.T) {
	usersTable := makeTable("users", 10432,
		[]map[string]any{
			makeColumn("id", "int4", false),
			makeColumn("email", "varchar(255)", false),
		},
		[]map[string]any{makePrimaryKeyIndex("users_pkey", []string{"id"})},
	)
	ordersTable := makeTable("orders", 182394,
		[]map[string]any{
			makeColumn("id", "int8", false),
			makeColumn("user_id", "int4", false),
		},
		[]map[string]any{makePrimaryKeyIndex("orders_pkey", []string{"id"})},
	)
	// Intentionally out of order to verify sorting.
	resp := makeMetadataResponse(makeSchemaWithViews("public", []string{"active_users", "recent_orders"}, ordersTable, usersTable))

	s, _ := mockMetadataServer(t, employeeDB(), resp)

	result, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	require.Equal(t, "POSTGRES", output.Engine)
	require.Len(t, output.Schemas, 1)
	section := output.Schemas[0]
	require.Equal(t, "public", section.Name)
	require.Len(t, section.Tables, 2)
	// Tables sorted alphabetically.
	require.Equal(t, "orders", section.Tables[0].Name)
	require.Equal(t, "users", section.Tables[1].Name)
	// Summary mode has no column details but includes column count.
	require.Equal(t, 2, section.Tables[0].ColumnCount)
	require.Equal(t, int64(182394), section.Tables[0].RowCount)
	require.Empty(t, section.Tables[0].Columns)
	// Views sorted alphabetically.
	require.Equal(t, []string{"active_users", "recent_orders"}, section.Views)

	// Text header mentions schema and tables.
	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Database: instances/prod-pg/databases/employee_db (POSTGRES)")
	require.Contains(t, text, "Schemas: 1 (public)")
	require.Contains(t, text, "Tables: 2")
}

func TestGetSchema_IncludeColumns(t *testing.T) {
	usersTable := makeTable("users", 10432,
		[]map[string]any{
			makeColumn("id", "int4", false),
			makeColumn("email", "varchar(255)", false),
			makeColumn("created_at", "timestamptz", true),
		},
		[]map[string]any{makePrimaryKeyIndex("users_pkey", []string{"id"})},
	)
	resp := makeMetadataResponse(makeSchema("public", usersTable))

	s, mock := mockMetadataServer(t, employeeDB(), resp)

	result, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Include:  "columns",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	users := output.Schemas[0].Tables[0]
	require.Len(t, users.Columns, 3)
	// Proto order preserved (id, email, created_at) — NOT alphabetized.
	require.Equal(t, "id", users.Columns[0].Name)
	require.Equal(t, "int4", users.Columns[0].Type)
	require.False(t, users.Columns[0].Nullable)
	require.True(t, users.Columns[0].PrimaryKey)
	require.Equal(t, "email", users.Columns[1].Name)
	require.False(t, users.Columns[1].PrimaryKey)
	require.Equal(t, "created_at", users.Columns[2].Name)
	require.True(t, users.Columns[2].Nullable)
	// columns mode has no defaults/comments populated.
	require.Empty(t, users.Columns[0].Default)
	require.Empty(t, users.Columns[0].Comment)
	// No indexes or FKs in columns mode.
	require.Empty(t, users.Indexes)
	require.Empty(t, users.ForeignKeys)

	// Request includes limit=201.
	require.Equal(t, int32(201), mock.lastCall().Limit)
}

func TestGetSchema_IncludeDetails(t *testing.T) {
	ordersTable := map[string]any{
		"name":     "orders",
		"rowCount": "182394",
		"comment":  "Customer orders",
		"columns": []map[string]any{
			makeColumnWithDefault("id", "int8", "", "Order ID"),
			makeColumnWithDefault("user_id", "int4", "", "Customer ref"),
			makeColumnWithDefault("status", "varchar(32)", "'pending'", "Status flag"),
		},
		"indexes": []map[string]any{
			// Intentionally out of order.
			makeIndex("orders_user_id_idx", false, []string{"user_id"}),
			makePrimaryKeyIndex("orders_pkey", []string{"id"}),
		},
		"foreignKeys": []map[string]any{
			{
				"name":              "orders_user_id_fkey",
				"columns":           []string{"user_id"},
				"referencedTable":   "users",
				"referencedColumns": []string{"id"},
			},
		},
	}
	resp := makeMetadataResponse(makeSchema("public", ordersTable))

	s, _ := mockMetadataServer(t, employeeDB(), resp)

	result, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Include:  "details",
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	orders := output.Schemas[0].Tables[0]
	require.Equal(t, "Customer orders", orders.Comment)
	// Column defaults and comments populated.
	require.Equal(t, "'pending'", orders.Columns[2].Default)
	require.Equal(t, "Status flag", orders.Columns[2].Comment)
	// Indexes sorted alphabetically.
	require.Len(t, orders.Indexes, 2)
	require.Equal(t, "orders_pkey", orders.Indexes[0].Name)
	require.Equal(t, "orders_user_id_idx", orders.Indexes[1].Name)
	// FKs present.
	require.Len(t, orders.ForeignKeys, 1)
	require.Equal(t, "orders_user_id_fkey", orders.ForeignKeys[0].Name)
	require.Equal(t, "users", orders.ForeignKeys[0].ReferencedTable)
}

func TestGetSchema_TableFilter_ServerSideFilter(t *testing.T) {
	ordersTable := makeTable("orders", 182394,
		[]map[string]any{
			makeColumn("id", "int8", false),
			makeColumn("user_id", "int4", false),
		},
		[]map[string]any{makePrimaryKeyIndex("orders_pkey", []string{"id"})},
	)
	resp := makeMetadataResponse(makeSchema("public", ordersTable))

	s, mock := mockMetadataServer(t, employeeDB(), resp)

	_, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Table:    "orders",
	})
	require.NoError(t, err)

	require.Equal(t, `table == "orders"`, mock.lastCall().Filter)
	// table= mode should NOT include a limit.
	require.Equal(t, int32(0), mock.lastCall().Limit)
}

func TestGetSchema_TableImpliesDetails(t *testing.T) {
	ordersTable := map[string]any{
		"name":     "orders",
		"rowCount": "182394",
		"comment":  "Customer orders",
		"columns": []map[string]any{
			makeColumnWithDefault("id", "int8", "", "Order ID"),
		},
		"indexes": []map[string]any{makePrimaryKeyIndex("orders_pkey", []string{"id"})},
		"foreignKeys": []map[string]any{
			{
				"name":              "orders_user_id_fkey",
				"columns":           []string{"user_id"},
				"referencedTable":   "users",
				"referencedColumns": []string{"id"},
			},
		},
	}
	resp := makeMetadataResponse(makeSchema("public", ordersTable))

	s, _ := mockMetadataServer(t, employeeDB(), resp)

	result, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Table:    "orders",
		// Include not set → should default to details.
	})
	require.NoError(t, err)
	require.False(t, result.IsError)

	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	require.NotNil(t, output.Table)
	// Details level: indexes and FKs are populated.
	require.NotEmpty(t, output.Table.Indexes)
	require.NotEmpty(t, output.Table.ForeignKeys)
	// Column comment (details-only field) populated.
	require.Equal(t, "Order ID", output.Table.Columns[0].Comment)
}

func TestGetSchema_TableNotFound_RefetchesForCandidates(t *testing.T) {
	// First call: server-side filter returns nothing.
	emptyResp := makeMetadataResponse(makeSchema("public"))
	// Second call: no filter, returns candidates.
	ordersTable := makeTable("orders", 0, []map[string]any{makeColumn("id", "int4", false)}, nil)
	orderItemsTable := makeTable("order_items", 0, []map[string]any{makeColumn("id", "int4", false)}, nil)
	unrelatedTable := makeTable("invoices", 0, []map[string]any{makeColumn("id", "int4", false)}, nil)
	fullResp := makeMetadataResponse(makeSchema("public", ordersTable, orderItemsTable, unrelatedTable))

	s, mock := mockMetadataServer(t, employeeDB(), emptyResp, fullResp)

	result, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Table:    "orderz",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	// Two metadata calls: filtered + candidate lookup.
	require.Equal(t, 2, mock.callCount())
	require.Equal(t, `table == "orderz"`, mock.callAt(0).Filter)
	require.Equal(t, "", mock.callAt(1).Filter)
	require.Equal(t, int32(schemaCandidateFetchLimit), mock.callAt(1).Limit)

	// Error body has candidates.
	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "TABLE_NOT_FOUND")
	require.Contains(t, text, "orders")
	require.Contains(t, text, "order_items")
	require.NotContains(t, text, "invoices")
}

func TestGetSchema_SchemaFilter_Postgres_ServerSideFilter(t *testing.T) {
	resp := makeMetadataResponse(makeSchema("public",
		makeTable("users", 10, []map[string]any{makeColumn("id", "int4", false)}, nil),
	))
	s, mock := mockMetadataServer(t, employeeDB(), resp)

	_, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Schema:   "public",
	})
	require.NoError(t, err)
	require.Equal(t, `schema == "public"`, mock.lastCall().Filter)
}

func TestGetSchema_SchemaAndTableFilter(t *testing.T) {
	resp := makeMetadataResponse(makeSchema("public",
		makeTable("orders", 10, []map[string]any{makeColumn("id", "int4", false)}, nil),
	))
	s, mock := mockMetadataServer(t, employeeDB(), resp)

	_, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Schema:   "public",
		Table:    "orders",
	})
	require.NoError(t, err)
	require.Equal(t, `schema == "public" && table == "orders"`, mock.lastCall().Filter)
}

func TestGetSchema_SchemaFilter_MySQL_DroppedClientSide(t *testing.T) {
	// MySQL has an empty schema name. The backend applies `schema == "X"` as an
	// exact match, which would filter out every table. The MCP layer must drop
	// the schema filter before calling the backend on single-schema engines.
	databases := []map[string]any{
		makeDatabase("instances/prod-mysql/databases/employee_db", "instances/prod-mysql", "projects/hr-system", "MYSQL", "ds-admin-1"),
	}
	resp := makeMetadataResponse(makeSchema("",
		makeTable("users", 10, []map[string]any{makeColumn("id", "int4", false)}, nil),
	))
	s, mock := mockMetadataServer(t, databases, resp)

	_, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Schema:   "anything",
	})
	require.NoError(t, err)
	// The schema filter is dropped client-side — the backend sees an empty filter.
	require.Equal(t, "", mock.lastCall().Filter)

	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	require.Len(t, output.Schemas, 1)
	require.Len(t, output.Schemas[0].Tables, 1)
}

func TestGetSchema_SchemaFilter_PassedThroughOnMultiSchemaEngines(t *testing.T) {
	// Spot-check a few known multi-schema engines to confirm the allowlist.
	for _, engine := range []string{"POSTGRES", "MSSQL", "ORACLE", "SNOWFLAKE", "REDSHIFT", "COCKROACHDB"} {
		t.Run(engine, func(t *testing.T) {
			databases := []map[string]any{
				makeDatabase("instances/prod/databases/employee_db", "instances/prod", "projects/hr-system", engine, "ds-1"),
			}
			resp := makeMetadataResponse(makeSchema("public",
				makeTable("users", 10, []map[string]any{makeColumn("id", "int4", false)}, nil),
			))
			s, mock := mockMetadataServer(t, databases, resp)

			_, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
				Database: "employee_db",
				Schema:   "public",
			})
			require.NoError(t, err)
			require.Equal(t, `schema == "public"`, mock.lastCall().Filter)
		})
	}
}

func TestGetSchema_SchemaFilter_DroppedOnSingleSchemaEngines(t *testing.T) {
	// Spot-check a few known single-schema engines.
	for _, engine := range []string{"MYSQL", "TIDB", "MARIADB", "CLICKHOUSE", "SQLITE", "SPANNER"} {
		t.Run(engine, func(t *testing.T) {
			databases := []map[string]any{
				makeDatabase("instances/prod/databases/employee_db", "instances/prod", "projects/hr-system", engine, "ds-1"),
			}
			resp := makeMetadataResponse(makeSchema("",
				makeTable("users", 10, []map[string]any{makeColumn("id", "int4", false)}, nil),
			))
			s, mock := mockMetadataServer(t, databases, resp)

			_, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
				Database: "employee_db",
				Schema:   "something",
			})
			require.NoError(t, err)
			require.Equal(t, "", mock.lastCall().Filter, "engine %s should drop the schema filter", engine)
		})
	}
}

func TestGetSchema_PrimaryKeyDerivation(t *testing.T) {
	usersTable := makeTable("users", 10,
		[]map[string]any{
			makeColumn("id", "int4", false),
			makeColumn("email", "varchar(255)", false),
			makeColumn("created_at", "timestamptz", false),
		},
		[]map[string]any{
			makePrimaryKeyIndex("users_pkey", []string{"id"}),
			makeIndex("users_email_idx", true, []string{"email"}), // unique but NOT primary
		},
	)
	resp := makeMetadataResponse(makeSchema("public", usersTable))
	s, _ := mockMetadataServer(t, employeeDB(), resp)

	_, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Include:  "columns",
	})
	require.NoError(t, err)
	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	cols := output.Schemas[0].Tables[0].Columns
	require.True(t, cols[0].PrimaryKey, "id should be a PK (primary index)")
	require.False(t, cols[1].PrimaryKey, "email should NOT be a PK (unique only)")
	require.False(t, cols[2].PrimaryKey, "created_at should NOT be a PK")
}

func TestGetSchema_LimitForBulkModes(t *testing.T) {
	resp := makeMetadataResponse(makeSchema("public",
		makeTable("users", 10, []map[string]any{makeColumn("id", "int4", false)}, nil),
	))

	// columns → limit=201
	s, mock := mockMetadataServer(t, employeeDB(), resp)
	_, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{Database: "employee_db", Include: "columns"})
	require.NoError(t, err)
	require.Equal(t, int32(201), mock.lastCall().Limit)

	// details → limit=201
	s2, mock2 := mockMetadataServer(t, employeeDB(), resp)
	_, _, err = s2.handleGetSchema(testContext(), nil, SchemaInput{Database: "employee_db", Include: "details"})
	require.NoError(t, err)
	require.Equal(t, int32(201), mock2.lastCall().Limit)

	// summary → no limit
	s3, mock3 := mockMetadataServer(t, employeeDB(), resp)
	_, _, err = s3.handleGetSchema(testContext(), nil, SchemaInput{Database: "employee_db"})
	require.NoError(t, err)
	require.Equal(t, int32(0), mock3.lastCall().Limit)

	// table= → no limit
	s4, mock4 := mockMetadataServer(t, employeeDB(), resp)
	_, _, err = s4.handleGetSchema(testContext(), nil, SchemaInput{Database: "employee_db", Table: "users"})
	require.NoError(t, err)
	require.Equal(t, int32(0), mock4.lastCall().Limit)
}

func TestGetSchema_NotTruncatedAtExactly200(t *testing.T) {
	tables := make([]map[string]any, schemaTableLimit)
	for i := range tables {
		tables[i] = makeTable(fmt.Sprintf("t%03d", i), 0, []map[string]any{makeColumn("id", "int4", false)}, nil)
	}
	resp := makeMetadataResponse(makeSchema("public", tables...))
	s, _ := mockMetadataServer(t, employeeDB(), resp)

	_, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Include:  "columns",
	})
	require.NoError(t, err)
	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	section := output.Schemas[0]
	require.Len(t, section.Tables, schemaTableLimit)
	require.False(t, section.Truncated)
	require.Equal(t, 0, section.TablesShown)
}

func TestGetSchema_TruncatedAt201(t *testing.T) {
	// Build 201 tables; the 201st sorted entry should be dropped.
	tables := make([]map[string]any, schemaTableLimit+1)
	for i := range tables {
		tables[i] = makeTable(fmt.Sprintf("t%03d", i), 0, []map[string]any{makeColumn("id", "int4", false)}, nil)
	}
	resp := makeMetadataResponse(makeSchema("public", tables...))
	s, _ := mockMetadataServer(t, employeeDB(), resp)

	_, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Include:  "columns",
	})
	require.NoError(t, err)
	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	section := output.Schemas[0]
	require.Len(t, section.Tables, schemaTableLimit)
	require.True(t, section.Truncated)
	require.Equal(t, schemaTableLimit, section.TablesShown)
	// The names are t000..t200; sorted alphabetically the 201st (last) is t200,
	// which should be dropped. Remaining last table is t199.
	require.Equal(t, "t199", section.Tables[schemaTableLimit-1].Name)
}

func TestGetSchema_PerSchemaLimitSemantics(t *testing.T) {
	// Build 3 schemas with 201 tables each (enough to trigger truncation in each).
	makeBigSchema := func(name string) map[string]any {
		tables := make([]map[string]any, schemaTableLimit+1)
		for i := range tables {
			tables[i] = makeTable(fmt.Sprintf("t%03d", i), 0, []map[string]any{makeColumn("id", "int4", false)}, nil)
		}
		return makeSchema(name, tables...)
	}
	resp := makeMetadataResponse(makeBigSchema("schema_a"), makeBigSchema("schema_b"), makeBigSchema("schema_c"))
	s, _ := mockMetadataServer(t, employeeDB(), resp)

	_, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Include:  "columns",
	})
	require.NoError(t, err)
	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	require.Len(t, output.Schemas, 3)
	totalTables := 0
	for _, section := range output.Schemas {
		require.True(t, section.Truncated)
		require.Len(t, section.Tables, schemaTableLimit)
		totalTables += len(section.Tables)
	}
	require.Equal(t, 3*schemaTableLimit, totalTables)
}

func TestGetSchema_DatabaseNotFound(t *testing.T) {
	s := newTestServerWithMock(t, mockListDatabases(employeeDB()))

	result, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "nonexistent",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "DATABASE_NOT_FOUND")
}

func TestGetSchema_AmbiguousDatabase(t *testing.T) {
	databases := []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr-system", "POSTGRES", "ds-admin-1"),
		makeDatabase("instances/staging-mysql/databases/employee_db", "instances/staging-mysql", "projects/hr-system", "MYSQL", "ds-admin-2"),
	}
	s := newTestServerWithMock(t, mockListDatabases(databases))

	result, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "AMBIGUOUS_TARGET")
}

func TestGetSchema_SchemaSyncFailed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "DatabaseService/GetDatabaseMetadata") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "sync timeout after 30s",
				"code":    "INTERNAL",
			})
			return
		}
		mockListDatabases(employeeDB()).ServeHTTP(w, r)
	})
	s := newTestServerWithMock(t, handler)

	result, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "SCHEMA_SYNC_FAILED")
	require.Contains(t, text, "sync timeout after 30s")
}

func TestGetSchema_PermissionDenied(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "DatabaseService/GetDatabaseMetadata") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "permission denied",
				"code":    "PERMISSION_DENIED",
			})
			return
		}
		mockListDatabases(employeeDB()).ServeHTTP(w, r)
	})
	s := newTestServerWithMock(t, handler)

	result, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "PERMISSION_DENIED")
	require.Contains(t, text, "bb.databases.getSchema")
}

func TestGetSchema_InvalidIncludeValue(t *testing.T) {
	s := newTestServerWithMock(t, http.NotFoundHandler())

	_, _, err := s.handleGetSchema(context.Background(), nil, SchemaInput{
		Database: "employee_db",
		Include:  "everything",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "summary|columns|details")
}

func TestGetSchema_MissingDatabase(t *testing.T) {
	s := newTestServerWithMock(t, http.NotFoundHandler())
	_, _, err := s.handleGetSchema(context.Background(), nil, SchemaInput{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "database is required")
}

func TestGetSchema_StableSortOrder(t *testing.T) {
	// Two schemas (out of order), each with a few tables (out of order).
	schemaB := makeSchema("schema_b",
		makeTable("zz", 0, []map[string]any{makeColumn("id", "int4", false)}, nil),
		makeTable("aa", 0, []map[string]any{makeColumn("id", "int4", false)}, nil),
	)
	schemaA := makeSchema("schema_a",
		makeTable("yy", 0, []map[string]any{makeColumn("id", "int4", false)}, nil),
		makeTable("bb", 0, []map[string]any{makeColumn("id", "int4", false)}, nil),
	)
	resp := makeMetadataResponse(schemaB, schemaA)

	s1, _ := mockMetadataServer(t, employeeDB(), resp)
	_, structured1, err := s1.handleGetSchema(testContext(), nil, SchemaInput{Database: "employee_db"})
	require.NoError(t, err)

	s2, _ := mockMetadataServer(t, employeeDB(), resp)
	_, structured2, err := s2.handleGetSchema(testContext(), nil, SchemaInput{Database: "employee_db"})
	require.NoError(t, err)

	json1, _ := json.Marshal(structured1)
	json2, _ := json.Marshal(structured2)
	require.Equal(t, string(json1), string(json2))

	// Explicit ordering assertions.
	output, ok := structured1.(*SchemaOutput)
	require.True(t, ok)
	require.Equal(t, "schema_a", output.Schemas[0].Name)
	require.Equal(t, "schema_b", output.Schemas[1].Name)
	require.Equal(t, "bb", output.Schemas[0].Tables[0].Name)
	require.Equal(t, "yy", output.Schemas[0].Tables[1].Name)
	require.Equal(t, "aa", output.Schemas[1].Tables[0].Name)
	require.Equal(t, "zz", output.Schemas[1].Tables[1].Name)
}

// TestGetSchema_SummaryDoesNotTruncate verifies the P1 fix: summary mode must
// return every table in a schema even when there are more than schemaTableLimit
// of them, because summary is designed to fit large catalogs (per-table payload
// is tiny) and the server isn't asked to limit in summary mode.
func TestGetSchema_SummaryDoesNotTruncate(t *testing.T) {
	// Build 250 tables — well over schemaTableLimit (200).
	tableCount := schemaTableLimit + 50
	tables := make([]map[string]any, tableCount)
	for i := range tables {
		tables[i] = makeTable(fmt.Sprintf("t%03d", i), 0, []map[string]any{makeColumn("id", "int4", false)}, nil)
	}
	resp := makeMetadataResponse(makeSchema("public", tables...))
	s, mock := mockMetadataServer(t, employeeDB(), resp)

	_, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		// No Include set → defaults to summary.
	})
	require.NoError(t, err)
	// Summary mode does not send a server-side limit.
	require.Equal(t, int32(0), mock.lastCall().Limit)

	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	require.Len(t, output.Schemas, 1)
	section := output.Schemas[0]
	require.Len(t, section.Tables, tableCount, "summary mode must not truncate")
	require.False(t, section.Truncated)
	require.Equal(t, 0, section.TablesShown)
}

// TestGetSchema_AmbiguousTableAcrossSchemas verifies the P2 fix: when the same
// table name exists in multiple schemas and the caller did not specify schema=,
// the tool must return an AMBIGUOUS_TABLE error instead of silently picking the
// first match.
func TestGetSchema_AmbiguousTableAcrossSchemas(t *testing.T) {
	usersPublic := makeTable("users", 10,
		[]map[string]any{makeColumn("id", "int4", false)},
		[]map[string]any{makePrimaryKeyIndex("users_pkey", []string{"id"})},
	)
	usersAnalytics := makeTable("users", 20,
		[]map[string]any{makeColumn("uid", "int4", false)},
		[]map[string]any{makePrimaryKeyIndex("users_pkey", []string{"uid"})},
	)
	resp := makeMetadataResponse(
		makeSchema("analytics", usersAnalytics),
		makeSchema("public", usersPublic),
	)
	s, _ := mockMetadataServer(t, employeeDB(), resp)

	result, _, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Table:    "users",
		// No Schema set → both matches are visible.
	})
	require.NoError(t, err)
	require.True(t, result.IsError)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "AMBIGUOUS_TABLE")
	require.Contains(t, text, "analytics")
	require.Contains(t, text, "public")
	require.Contains(t, text, "schema=")
}

// TestGetSchema_TableInSpecificSchema verifies the happy path for the P2 fix:
// with a schema= hint, a single match proceeds normally.
func TestGetSchema_TableInSpecificSchema(t *testing.T) {
	usersPublic := makeTable("users", 10,
		[]map[string]any{makeColumn("id", "int4", false)},
		[]map[string]any{makePrimaryKeyIndex("users_pkey", []string{"id"})},
	)
	// Simulate a server that honors the schema filter: only `public` is returned.
	resp := makeMetadataResponse(makeSchema("public", usersPublic))
	s, mock := mockMetadataServer(t, employeeDB(), resp)

	_, structured, err := s.handleGetSchema(testContext(), nil, SchemaInput{
		Database: "employee_db",
		Schema:   "public",
		Table:    "users",
	})
	require.NoError(t, err)
	require.Equal(t, `schema == "public" && table == "users"`, mock.lastCall().Filter)

	output, ok := structured.(*SchemaOutput)
	require.True(t, ok)
	require.NotNil(t, output.Table)
	require.Equal(t, "users", output.Table.Name)
}

func TestBuildMetadataFilter(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		table  string
		want   string
	}{
		{"empty", "", "", ""},
		{"schema only", "public", "", `schema == "public"`},
		{"table only", "", "orders", `table == "orders"`},
		{"both", "public", "orders", `schema == "public" && table == "orders"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, buildMetadataFilter(tc.schema, tc.table))
		})
	}
}
