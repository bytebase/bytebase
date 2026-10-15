package aireview

import (
	"context"
	jsonv2 "encoding/json/v2"
	"fmt"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	// The tools render definitions with the engines' renderers.
	// The renderers the server registers, so every case runs on what production uses.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/clickhouse"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/trino"
)

// shopSchema is a database with a large table, the objects that depend on it,
// and a second table of the same name in another schema. An engine without
// schemas passes "" for both names and gets one schema.
func shopSchema(schemaName string, archiveName string, trigger *metadatapb.TriggerMetadata) *metadatapb.DatabaseSchemaMetadata {
	schemas := []*metadatapb.SchemaMetadata{{
		Name: schemaName,
		Tables: []*metadatapb.TableMetadata{
			{
				Name: "orders",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "bigint"},
					{Name: "customer_id", Type: "bigint"},
					{Name: "legacy_status", Type: "text", Nullable: true, Comment: "replaced by status"},
				},
				Indexes: []*metadatapb.IndexMetadata{
					{Name: "orders_pkey", Expressions: []string{"id"}, Type: "btree", Unique: true, Primary: true, IsConstraint: true, Visible: true},
					{Name: "idx_orders_customer", Expressions: []string{"customer_id"}, Type: "btree", Visible: true},
				},
				ForeignKeys: []*metadatapb.ForeignKeyMetadata{{
					Name:              "orders_customer_fk",
					Columns:           []string{"customer_id"},
					ReferencedSchema:  schemaName,
					ReferencedTable:   "customers",
					ReferencedColumns: []string{"id"},
				}},
				Triggers:  []*metadatapb.TriggerMetadata{trigger},
				Comment:   "one row per order",
				RowCount:  52000000,
				DataSize:  9 << 30,
				IndexSize: 1 << 30,
			},
			{
				Name:    "customers",
				Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}},
				Indexes: []*metadatapb.IndexMetadata{
					{Name: "customers_pkey", Expressions: []string{"id"}, Type: "btree", Unique: true, Primary: true, IsConstraint: true, Visible: true},
				},
				RowCount: 1200,
			},
		},
		Views: []*metadatapb.ViewMetadata{{
			Name:       "revenue_by_status",
			Definition: "SELECT legacy_status, count(*) AS order_count FROM orders GROUP BY legacy_status",
		}},
		Functions: []*metadatapb.FunctionMetadata{{
			Name:       "audit_change",
			Definition: "CREATE FUNCTION audit_change() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO change_log VALUES (NEW.id); RETURN NEW; END $$",
		}},
		Sequences: []*metadatapb.SequenceMetadata{{
			Name: "orders_id_seq", DataType: "bigint", Start: "1", Increment: "1", MinValue: "1", MaxValue: "9223372036854775807", CacheSize: "1",
			OwnerTable: "orders", OwnerColumn: "id",
		}},
		EnumTypes: []*metadatapb.EnumTypeMetadata{{Name: "payment_state", Values: []string{"pending", "paid"}}},
	}}
	if archiveName != "" {
		schemas = append(schemas, &metadatapb.SchemaMetadata{
			Name: archiveName,
			Tables: []*metadatapb.TableMetadata{{
				Name:    "orders",
				Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}},
			}},
		})
	}
	return &metadatapb.DatabaseSchemaMetadata{Name: "shop", Schemas: schemas}
}

func pgShop() *catalogTools {
	return &catalogTools{engine: storepb.Engine_POSTGRES, schema: shopSchema("public", "archive", &metadatapb.TriggerMetadata{
		Name: "orders_audit",
		Body: "CREATE TRIGGER orders_audit AFTER UPDATE ON public.orders FOR EACH ROW EXECUTE FUNCTION audit_change()",
	})}
}

func mysqlShop() Tools {
	return NewCatalogTools(storepb.Engine_MYSQL, shopSchema("", "", &metadatapb.TriggerMetadata{
		Name:   "orders_audit",
		Timing: "AFTER",
		Event:  "UPDATE",
		Body:   "INSERT INTO change_log (order_id) VALUES (NEW.id)",
	}))
}

func search(t *testing.T, tools Tools, arguments string) *searchResult {
	t.Helper()
	output, err := tools.Call(context.Background(), toolSearch, arguments)
	require.NoError(t, err)
	result := &searchResult{}
	require.NoError(t, jsonv2.Unmarshal([]byte(output), result), output)
	return result
}

func read(t *testing.T, tools Tools, arguments string) *readResult {
	t.Helper()
	output, err := tools.Call(context.Background(), toolRead, arguments)
	require.NoError(t, err)
	result := &readResult{}
	require.NoError(t, jsonv2.Unmarshal([]byte(output), result), output)
	return result
}

func TestCatalogDefinitions(t *testing.T) {
	t.Parallel()

	definitions := pgShop().Definitions()
	require.Len(t, definitions, 2)
	require.Equal(t, toolSearch, definitions[0].GetName())
	require.Equal(t, toolRead, definitions[1].GetName())
	for _, definition := range definitions {
		var parameters map[string]any
		require.NoError(t, jsonv2.Unmarshal([]byte(definition.GetParametersSchema()), &parameters), definition.GetName())
		require.Equal(t, "object", parameters["type"])
	}
}

func TestCatalogSearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments string
		want      []searchMatch
	}{
		{
			name:      "a column is found in the definitions that use it",
			arguments: `{"text": "legacy_status"}`,
			want: []searchMatch{
				{Kind: kindTable, Schema: "public", Name: "orders", Matched: "definition"},
				{Kind: kindView, Schema: "public", Name: "revenue_by_status", Matched: "definition"},
			},
		},
		{
			name:      "a function is found through the trigger that calls it",
			arguments: `{"text": "AUDIT_CHANGE"}`,
			want: []searchMatch{
				{Kind: kindTable, Schema: "public", Name: "orders", Matched: "definition"},
				{Kind: kindFunction, Schema: "public", Name: "audit_change", Matched: "name"},
			},
		},
		{
			name:      "a table is found by the foreign key that references it",
			arguments: `{"text": "customers"}`,
			want: []searchMatch{
				{Kind: kindTable, Schema: "public", Name: "orders", Matched: "definition"},
				{Kind: kindTable, Schema: "public", Name: "customers", Matched: "name"},
			},
		},
		{
			name:      "the schema limits the search",
			arguments: `{"text": "orders", "schema": "archive"}`,
			want:      []searchMatch{{Kind: kindTable, Schema: "archive", Name: "orders", Matched: "name"}},
		},
		{
			name:      "an enum type is found by a value",
			arguments: `{"text": "pending"}`,
			want:      []searchMatch{{Kind: kindEnumType, Schema: "public", Name: "payment_state", Matched: "definition"}},
		},
		{
			name:      "nothing matches",
			arguments: `{"text": "refunds"}`,
			want:      []searchMatch{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := search(t, pgShop(), test.arguments)
			require.Equal(t, test.want, result.Matches)
			require.Equal(t, len(test.want), result.Total)
			require.Nil(t, result.NextFrom)
		})
	}

	everything := search(t, pgShop(), `{"text": ""}`)
	require.Equal(t, 7, everything.Total, "an empty text lists every object")

	output, err := pgShop().Call(context.Background(), toolSearch, `{"text": "refunds"}`)
	require.NoError(t, err)
	require.JSONEq(t, `{"matches": [], "total": 0}`, output, "no match is an empty list, never null")
}

func TestCatalogReadPostgres(t *testing.T) {
	t.Parallel()

	result := read(t, pgShop(), `{"objects": [{"schema": "public", "name": "orders"}, {"schema": "public", "name": "customers"}, {"schema": "public", "name": "revenue_by_status"}, {"schema": "public", "name": "refunds"}]}`)
	require.Len(t, result.Objects, 3)
	require.Equal(t, []objectName{{Schema: "public", Name: "refunds"}}, result.NotFound)
	require.Nil(t, result.NextFrom)

	orders := result.Objects[0]
	require.Equal(t, kindTable, orders.Kind)
	require.Equal(t, statistics(52000000, 9<<30, 1<<30), orders.Statistics)
	for _, part := range []string{
		`CREATE TABLE "public"."orders"`,
		`"legacy_status" text`,
		`COMMENT ON COLUMN "public"."orders"."legacy_status" IS 'replaced by status'`,
		`ADD CONSTRAINT "orders_pkey" PRIMARY KEY (id)`,
		`CREATE INDEX "idx_orders_customer"`,
		`ADD CONSTRAINT "orders_customer_fk" FOREIGN KEY ("customer_id")`,
		`ALTER SEQUENCE "public"."orders_id_seq" OWNED BY "public"."orders"."id"`,
	} {
		require.Contains(t, orders.Definition, part)
	}
	require.Equal(t, 1, strings.Count(orders.Definition, "CREATE TRIGGER orders_audit"), "the renderer wrote the trigger, so it is not added again")

	customers := result.Objects[1]
	require.Equal(t, statistics(1200, 0, 0), customers.Statistics)
	archived := read(t, pgShop(), `{"objects": [{"schema": "archive", "name": "orders"}]}`).Objects[0]
	require.Equal(t, statistics(0, 0, 0), archived.Statistics, "PostgreSQL syncs every statistic, so the zeros it read are shown")
	require.NotContains(t, customers.Definition, "orders_id_seq", "a table shows only the sequences it owns")

	view := result.Objects[2]
	require.Equal(t, kindView, view.Kind)
	require.Nil(t, view.Statistics)
	require.Contains(t, view.Definition, `CREATE VIEW "public"."revenue_by_status" AS`)
}

func TestCatalogReadLooksUpNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments string
		want      []objectName
	}{
		{
			name:      "a name without a schema returns every object of that name",
			arguments: `{"objects": [{"name": "orders"}]}`,
			want:      []objectName{{Schema: "public", Name: "orders"}, {Schema: "archive", Name: "orders"}},
		},
		{
			name:      "a name that differs only in case is found",
			arguments: `{"objects": [{"schema": "PUBLIC", "name": "ORDERS"}]}`,
			want:      []objectName{{Schema: "public", Name: "orders"}},
		},
		{
			name:      "an object named twice is returned once",
			arguments: `{"objects": [{"schema": "public", "name": "orders"}, {"name": "orders"}]}`,
			want:      []objectName{{Schema: "public", Name: "orders"}, {Schema: "archive", Name: "orders"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := read(t, pgShop(), test.arguments)
			var got []objectName
			for _, object := range result.Objects {
				got = append(got, objectName{Schema: object.Schema, Name: object.Name})
			}
			require.Equal(t, test.want, got)
			require.Empty(t, result.NotFound)
		})
	}
}

func TestCatalogReadMySQL(t *testing.T) {
	t.Parallel()

	// MySQL has no schemas, and the model tends to pass the database name.
	result := read(t, mysqlShop(), `{"objects": [{"schema": "shop", "name": "orders"}]}`)
	require.Len(t, result.Objects, 1)
	orders := result.Objects[0]
	require.Empty(t, orders.Schema)
	require.Equal(t, statistics(52000000, 9<<30, 1<<30), orders.Statistics)
	for _, part := range []string{
		"CREATE TABLE `orders`",
		"`legacy_status` text COMMENT 'replaced by status'",
		"PRIMARY KEY (`id`)",
		"KEY `idx_orders_customer` (`customer_id`)",
		"CONSTRAINT `orders_customer_fk` FOREIGN KEY (`customer_id`) REFERENCES `customers` (`id`)",
		// The MySQL renderer leaves the triggers out.
		"TRIGGER orders_audit AFTER UPDATE ON orders\nINSERT INTO change_log (order_id) VALUES (NEW.id)",
	} {
		require.Contains(t, orders.Definition, part)
	}

	matches := search(t, mysqlShop(), `{"text": "change_log", "schema": "shop"}`)
	require.Equal(t, []searchMatch{
		{Kind: kindTable, Name: "orders", Matched: "definition"},
		{Kind: kindFunction, Name: "audit_change", Matched: "definition"},
	}, matches.Matches)
}

func TestCatalogReadWithoutARenderer(t *testing.T) {
	t.Parallel()

	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, shopSchema("PUBLIC", "", &metadatapb.TriggerMetadata{}))
	result := read(t, tools, `{"objects": [{"name": "orders"}, {"name": "revenue_by_status"}, {"name": "orders_id_seq"}, {"name": "payment_state"}]}`)
	require.Len(t, result.Objects, 4)
	require.Equal(t, strings.Join([]string{
		"TABLE PUBLIC.orders (",
		"  id bigint NOT NULL",
		"  customer_id bigint NOT NULL",
		"  legacy_status text",
		")",
		"",
		"PRIMARY KEY orders_pkey USING btree (id)",
		"INDEX idx_orders_customer USING btree (customer_id)",
		"",
		"FOREIGN KEY orders_customer_fk (customer_id) REFERENCES PUBLIC.customers (id)",
		"",
		"COMMENT: one row per order",
		"",
		"COMMENT ON COLUMN legacy_status: replaced by status",
	}, "\n"), result.Objects[0].Definition)
	require.Equal(t, "VIEW PUBLIC.revenue_by_status AS\nSELECT legacy_status, count(*) AS order_count FROM orders GROUP BY legacy_status", result.Objects[1].Definition)
	require.Equal(t, "SEQUENCE PUBLIC.orders_id_seq AS bigint START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1 OWNED BY orders.id", result.Objects[2].Definition)
	require.Equal(t, "ENUM TYPE PUBLIC.payment_state: 'pending', 'paid'", result.Objects[3].Definition)
}

func TestCatalogSearchPages(t *testing.T) {
	t.Parallel()

	const tableCount = 2000
	schema := &metadatapb.SchemaMetadata{Name: "public"}
	for i := range tableCount {
		schema.Tables = append(schema.Tables, &metadatapb.TableMetadata{
			Name:    fmt.Sprintf("event_partition_with_a_long_name_%04d", i),
			Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}},
		})
	}
	tools := unbounded(NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{schema}}))

	var names []string
	from, pages := 0, 0
	for {
		output, err := tools.Call(context.Background(), toolSearch, fmt.Sprintf(`{"text": "event_partition", "from": %d}`, from))
		require.NoError(t, err)
		require.Less(t, len(output), maxToolResultBytes+1024)
		result := &searchResult{}
		require.NoError(t, jsonv2.Unmarshal([]byte(output), result))
		require.Equal(t, tableCount, result.Total)
		require.NotEmpty(t, result.Matches)
		for _, match := range result.Matches {
			names = append(names, match.Name)
		}
		pages++
		if result.NextFrom == nil {
			require.Empty(t, result.Note)
			break
		}
		require.Equal(t, from+len(result.Matches), *result.NextFrom)
		require.Contains(t, result.Note, fmt.Sprintf("from set to %d", *result.NextFrom))
		from = *result.NextFrom
	}
	require.Greater(t, pages, 1)
	require.Len(t, names, tableCount)
	for i, name := range names {
		require.Equal(t, fmt.Sprintf("event_partition_with_a_long_name_%04d", i), name)
	}

	past := search(t, tools, fmt.Sprintf(`{"text": "event_partition", "from": %d}`, tableCount))
	require.Empty(t, past.Matches)
	require.Equal(t, tableCount, past.Total)
}

// readAll follows next_from until the result is whole and returns the text of
// every object read, in order, keyed by signature or name.
func readAll(t *testing.T, tools Tools, objects string) ([]string, map[string]string) {
	t.Helper()
	var order []string
	text := make(map[string]string)
	from := 0
	for calls := 1; ; calls++ {
		require.Less(t, calls, 500, "the pages must end")
		output, err := tools.Call(context.Background(), toolRead, fmt.Sprintf(`{"objects": %s, "from": %d}`, objects, from))
		require.NoError(t, err)
		require.Less(t, len(output), maxToolResultBytes+1024)
		result := &readResult{}
		require.NoError(t, jsonv2.Unmarshal([]byte(output), result), output)
		require.NotEmpty(t, result.Objects, "every call makes progress")
		for _, object := range result.Objects {
			require.True(t, utf8.ValidString(object.Definition), "a definition is never cut inside a character")
			key := object.Schema + "." + object.Name + object.Signature
			if _, ok := text[key]; !ok {
				order = append(order, key)
			}
			text[key] += object.Definition
		}
		if result.NextFrom == nil {
			require.Empty(t, result.Note)
			return order, text
		}
		require.Greater(t, *result.NextFrom, from)
		require.Equal(t, fmt.Sprintf("the result was cut; call read again with the same objects and from set to %d", *result.NextFrom), result.Note)
		from = *result.NextFrom
	}
}

func TestCatalogReadCutsABatch(t *testing.T) {
	t.Parallel()

	const viewCount = 20
	schema := &metadatapb.SchemaMetadata{Name: "PUBLIC"}
	var names []objectName
	var want []string
	for i := range viewCount {
		name := fmt.Sprintf("report_%02d", i)
		schema.Views = append(schema.Views, &metadatapb.ViewMetadata{
			Name:       name,
			Definition: "SELECT " + strings.Repeat("amount, ", 500) + "id FROM orders",
		})
		names = append(names, objectName{Schema: "PUBLIC", Name: name})
		want = append(want, "PUBLIC."+name)
	}
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{schema}})
	objects, err := jsonv2.Marshal(names)
	require.NoError(t, err)

	first := read(t, tools, fmt.Sprintf(`{"objects": %s}`, objects))
	require.Greater(t, len(first.Objects), 1)
	require.Less(t, len(first.Objects), viewCount)
	for _, object := range first.Objects {
		require.Empty(t, object.Part, "a definition that fits a result of its own is never cut to fill one")
	}

	order, text := readAll(t, tools, string(objects))
	require.Equal(t, want, order)
	for _, name := range want {
		require.Equal(t, "VIEW "+name+" AS\nSELECT "+strings.Repeat("amount, ", 500)+"id FROM orders", text[name])
	}
}

func TestCatalogReadPagesOneDefinition(t *testing.T) {
	t.Parallel()

	lines := []string{"CREATE FUNCTION settle() RETURNS void LANGUAGE plpgsql AS $$", "BEGIN"}
	for i := range 3000 {
		lines = append(lines, fmt.Sprintf("  UPDATE orders SET total = total + %d WHERE id = %d;", i, i))
	}
	lines = append(lines, "END $$")
	definition := strings.Join(lines, "\n")
	tools := unbounded(NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:      "PUBLIC",
		Functions: []*metadatapb.FunctionMetadata{{Name: "settle", Definition: definition}},
		Views:     []*metadatapb.ViewMetadata{{Name: "settled", Definition: "SELECT 1"}},
	}}}))

	first := read(t, tools, `{"objects": [{"name": "settle"}, {"name": "settled"}]}`)
	require.Len(t, first.Objects, 1, "the large definition takes the whole first result")
	piece := first.Objects[0]
	require.True(t, strings.HasSuffix(piece.Definition, ";\n"), "the cut falls behind a line break")
	require.Equal(t, fmt.Sprintf("bytes 0 to %d of %d", len(piece.Definition), len(definition)), piece.Part)
	require.Equal(t, len(piece.Definition), *first.NextFrom)

	order, text := readAll(t, tools, `[{"name": "settle"}, {"name": "settled"}]`)
	require.Equal(t, []string{"PUBLIC.settle", "PUBLIC.settled"}, order)
	require.Equal(t, definition, text["PUBLIC.settle"])
	require.Equal(t, "VIEW PUBLIC.settled AS\nSELECT 1", text["PUBLIC.settled"])

	// Every definition takes one position more than its bytes.
	total := len(definition) + 1 + len(text["PUBLIC.settled"]) + 1
	past := read(t, tools, fmt.Sprintf(`{"objects": [{"name": "settle"}, {"name": "settled"}], "from": %d}`, total))
	require.Empty(t, past.Objects)
	require.Equal(t, fmt.Sprintf("the result ends at %d, so nothing starts at %d", total, total), past.Note)
}

func TestCatalogReadMovesPastAnEmptyDefinition(t *testing.T) {
	t.Parallel()

	large := "SELECT '" + strings.Repeat("x", 2*maxToolResultBytes) + "'"
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "PUBLIC",
		Functions: []*metadatapb.FunctionMetadata{
			{Name: "empty_first"},
			{Name: "large", Definition: large},
			{Name: "empty_last"},
		},
	}}})

	// The empty definition fits and the large one does not, so the cursor
	// stops between the two. It has to stand behind the empty one.
	first := read(t, tools, `{"objects": [{"name": "empty_first"}, {"name": "large"}, {"name": "empty_last"}]}`)
	require.Len(t, first.Objects, 1)
	require.Equal(t, "empty_first", first.Objects[0].Name)
	require.Equal(t, 1, *first.NextFrom)

	order, text := readAll(t, tools, `[{"name": "empty_first"}, {"name": "large"}, {"name": "empty_last"}]`)
	require.Equal(t, []string{"PUBLIC.empty_first", "PUBLIC.large", "PUBLIC.empty_last"}, order)
	require.Equal(t, map[string]string{"PUBLIC.empty_first": "", "PUBLIC.large": large, "PUBLIC.empty_last": ""}, text)
}

func TestCatalogShowsEveryComment(t *testing.T) {
	t.Parallel()

	database := func() *metadatapb.DatabaseSchemaMetadata {
		return &metadatapb.DatabaseSchemaMetadata{
			Schemas: []*metadatapb.SchemaMetadata{{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{{
					Name:     "orders",
					Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint", Comment: "comment of the column"}},
					Indexes:  []*metadatapb.IndexMetadata{{Name: "idx_orders", Expressions: []string{"id"}, Type: "btree", Visible: true, Comment: "comment of the index"}},
					Triggers: []*metadatapb.TriggerMetadata{{Name: "audit", Timing: "AFTER", Event: "UPDATE", Body: "CREATE TRIGGER audit AFTER UPDATE ON public.orders FOR EACH ROW EXECUTE FUNCTION audit()", Comment: "comment of the trigger"}},
					Comment:  "comment of the table",
				}},
				Views:             []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT 1", Comment: "comment of the view"}},
				MaterializedViews: []*metadatapb.MaterializedViewMetadata{{Name: "mv", Definition: "SELECT 1", Comment: "comment of the materialized view"}},
				Functions:         []*metadatapb.FunctionMetadata{{Name: "f", Definition: "CREATE FUNCTION f() RETURNS int AS $$ SELECT 1 $$", Comment: "comment of the function"}},
				Procedures:        []*metadatapb.ProcedureMetadata{{Name: "p", Definition: "CREATE PROCEDURE p() AS $$ SELECT 1 $$", Comment: "comment of the procedure"}},
				Sequences:         []*metadatapb.SequenceMetadata{{Name: "s", DataType: "bigint", Start: "1", Increment: "1", MinValue: "1", MaxValue: "10", CacheSize: "1", Comment: "comment of the sequence"}},
				Events:            []*metadatapb.EventMetadata{{Name: "e", Definition: "CREATE EVENT e", Comment: "comment of the scheduled event"}},
				Streams:           []*metadatapb.StreamMetadata{{Name: "st", Definition: "CREATE STREAM st", Comment: "comment of the stream"}},
				Tasks:             []*metadatapb.TaskMetadata{{Name: "t", Definition: "CREATE TASK t", Comment: "comment of the task"}},
				EnumTypes:         []*metadatapb.EnumTypeMetadata{{Name: "en", Values: []string{"a"}, Comment: "comment of the enum type"}},
				CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
					Name:       "ct",
					Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "a", Type: "int", Comment: "comment of the attribute"}},
					Comment:    "comment of the composite type",
				}},
			}},
			EventTriggers: []*metadatapb.EventTriggerMetadata{{Name: "et", Event: "sql_drop", Enabled: true, FunctionName: "refuse", Comment: "comment of the event trigger"}},
		}
	}
	comments := []string{
		"column", "index", "trigger", "table", "view", "materialized view", "function", "procedure", "sequence",
		"scheduled event", "stream", "task", "enum type", "attribute", "composite type", "event trigger",
	}
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_SNOWFLAKE} {
		tools := NewCatalogTools(engine, database())
		for _, comment := range comments {
			text := "comment of the " + comment
			matches := search(t, tools, fmt.Sprintf(`{"text": %q}`, text)).Matches
			require.Len(t, matches, 1, "%s: %s", engine, text)
			definition := read(t, tools, fmt.Sprintf(`{"objects": [{"name": %q}]}`, matches[0].Name)).Objects[0].Definition
			require.Regexp(t, "COMMENT[^\n]*"+text, definition, "%s: the text is marked as a comment", engine)
			require.Equal(t, 1, strings.Count(definition, text), "%s: %s is written once", engine, text)
		}
	}
}

func TestCatalogMarksACommentThatRepeatsAName(t *testing.T) {
	t.Parallel()

	// Each comment is a text the definition holds for another reason.
	database := func() *metadatapb.DatabaseSchemaMetadata {
		return &metadatapb.DatabaseSchemaMetadata{
			Schemas: []*metadatapb.SchemaMetadata{{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{{
					Name:    "orders",
					Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}},
					Indexes: []*metadatapb.IndexMetadata{{Name: "idx_orders", Expressions: []string{"id"}, Type: "btree", Visible: true, Comment: "id"}},
					Comment: "orders",
				}},
			}},
			EventTriggers: []*metadatapb.EventTriggerMetadata{{Name: "block_drop", Event: "sql_drop", Enabled: true, FunctionName: "refuse", Comment: "sql_drop"}},
		}
	}
	tests := []struct {
		engine storepb.Engine
		object string
		want   []string
	}{
		{engine: storepb.Engine_POSTGRES, object: "orders", want: []string{`COMMENT ON TABLE "public"."orders" IS 'orders';`, `COMMENT ON INDEX "public"."idx_orders" IS 'id';`}},
		{engine: storepb.Engine_MYSQL, object: "orders", want: []string{"COMMENT='orders'", "COMMENT ON INDEX idx_orders: id"}},
		{engine: storepb.Engine_SNOWFLAKE, object: "orders", want: []string{"COMMENT: orders", "COMMENT ON INDEX idx_orders: id"}},
		{engine: storepb.Engine_POSTGRES, object: "block_drop", want: []string{"COMMENT: sql_drop"}},
	}
	for _, test := range tests {
		definition := read(t, NewCatalogTools(test.engine, database()), fmt.Sprintf(`{"objects": [{"name": %q}]}`, test.object)).Objects[0].Definition
		for _, want := range test.want {
			require.Equal(t, 1, strings.Count(definition, want), "%s %s: %q in\n%s", test.engine, test.object, want, definition)
		}
		require.Equal(t, len(test.want), strings.Count(definition, "COMMENT"), "%s %s: no comment twice in\n%s", test.engine, test.object, definition)
	}
}

func TestCatalogAddsWhatADumpLeavesOut(t *testing.T) {
	t.Parallel()

	trigger := func(name string, skipDump bool) *metadatapb.TriggerMetadata {
		return &metadatapb.TriggerMetadata{
			Name:     name,
			Body:     fmt.Sprintf("CREATE TRIGGER %s AFTER UPDATE ON public.orders FOR EACH ROW EXECUTE FUNCTION %s()", name, name),
			Comment:  "comment of " + name,
			SkipDump: skipDump,
		}
	}
	tools := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "public",
		Tables: []*metadatapb.TableMetadata{
			{
				// An extension owns the second trigger, so a dump skips it.
				Name:     "orders",
				Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}},
				Triggers: []*metadatapb.TriggerMetadata{trigger("audit", false), trigger("ts_insert_blocker", true)},
			},
			{
				// The sync user cannot see the columns, and the renderer writes
				// a bare table for it.
				Name:             "payments",
				Indexes:          []*metadatapb.IndexMetadata{{Name: "idx_payments_order", Expressions: []string{"order_id"}, Type: "btree", Definition: "CREATE INDEX idx_payments_order ON public.payments USING btree (order_id)", Comment: "comment of the index"}},
				ForeignKeys:      []*metadatapb.ForeignKeyMetadata{{Name: "payments_order_fk", Columns: []string{"order_id"}, ReferencedSchema: "public", ReferencedTable: "orders", ReferencedColumns: []string{"id"}}},
				CheckConstraints: []*metadatapb.CheckConstraintMetadata{{Name: "payments_amount_check", Expression: "(amount > 0)"}},
				Rules:            []*metadatapb.RuleMetadata{{Name: "keep_payments", Definition: "CREATE RULE keep_payments AS ON DELETE TO public.payments DO INSTEAD NOTHING"}},
				Triggers:         []*metadatapb.TriggerMetadata{trigger("audit_payments", false)},
				Partitions: []*metadatapb.TablePartitionMetadata{{
					Name:       "payments_2026",
					Type:       metadatapb.TablePartitionMetadata_RANGE,
					Expression: "paid_at",
					Value:      "FOR VALUES FROM ('2026-01-01') TO ('2027-01-01')",
					Indexes:    []*metadatapb.IndexMetadata{{Name: "payments_2026_order_idx", Expressions: []string{"order_id"}, Type: "btree", Comment: "comment of the partition index"}},
					Subpartitions: []*metadatapb.TablePartitionMetadata{{
						Name:             "payments_2026_eu",
						Type:             metadatapb.TablePartitionMetadata_LIST,
						Expression:       "region",
						Value:            "FOR VALUES IN ('eu')",
						CheckConstraints: []*metadatapb.CheckConstraintMetadata{{Name: "payments_2026_eu_check", Expression: "(region = 'eu')"}},
					}},
				}},
			},
		},
	}}})

	orders := read(t, tools, `{"objects": [{"name": "orders"}]}`).Objects[0].Definition
	for _, want := range []string{
		"CREATE TRIGGER audit AFTER UPDATE",
		`COMMENT ON TRIGGER "audit" ON "public"."orders" IS 'comment of audit'`,
		"CREATE TRIGGER ts_insert_blocker AFTER UPDATE",
		"COMMENT ON TRIGGER ts_insert_blocker: comment of ts_insert_blocker",
	} {
		require.Equal(t, 1, strings.Count(orders, want), "%q in\n%s", want, orders)
	}

	payments := read(t, tools, `{"objects": [{"name": "payments"}]}`).Objects[0].Definition
	for _, want := range []string{
		"CREATE INDEX idx_payments_order ON public.payments USING btree (order_id)",
		"COMMENT ON INDEX idx_payments_order: comment of the index",
		"FOREIGN KEY payments_order_fk (order_id) REFERENCES public.orders (id)",
		"payments_amount_check",
		"CREATE RULE keep_payments",
		"CREATE TRIGGER audit_payments AFTER UPDATE",
		"comment of audit_payments",
		"PARTITION payments_2026 BY RANGE (paid_at) VALUES FOR VALUES FROM ('2026-01-01') TO ('2027-01-01')",
		"  PARTITION payments_2026_eu BY LIST (region) VALUES FOR VALUES IN ('eu')",
		"ON PARTITION payments_2026:\n  INDEX payments_2026_order_idx USING btree (order_id)",
		"ON PARTITION payments_2026:\n  COMMENT ON INDEX payments_2026_order_idx: comment of the partition index",
		"ON PARTITION payments_2026_eu:\n  CHECK payments_2026_eu_check (region = 'eu')",
	} {
		require.Equal(t, 1, strings.Count(payments, want), "%q in\n%s", want, payments)
	}
}

func TestCatalogSearchKeepsTheMatchesForTheNextPage(t *testing.T) {
	t.Parallel()

	tools := pgShop()
	first := search(t, tools, `{"text": "orders"}`)
	kept := tools.matches
	require.Len(t, kept, first.Total)

	// The next page of the same search reads the matches kept. Clearing the
	// index shows that it does not scan again.
	tools.objects = nil
	require.Equal(t, first.Matches[1:], search(t, tools, `{"text": "orders", "from": 1}`).Matches)
	require.Empty(t, search(t, tools, `{"text": "customers"}`).Matches, "another search scans again")
}

func TestCatalogReadPagesALineOverTheLimit(t *testing.T) {
	t.Parallel()

	// A character of three bytes puts the limit inside a character.
	definition := "SELECT '" + strings.Repeat("数", 100000) + "'"
	tools := unbounded(NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:      "PUBLIC",
		Functions: []*metadatapb.FunctionMetadata{{Name: "minified", Definition: definition}},
	}}}))

	order, text := readAll(t, tools, `[{"name": "minified"}]`)
	require.Equal(t, []string{"PUBLIC.minified"}, order)
	require.Equal(t, definition, text["PUBLIC.minified"])

	// A from of the model's own making can point inside a character, here the
	// last one before the closing quote. The result starts at the next one.
	inside := read(t, tools, fmt.Sprintf(`{"objects": [{"name": "minified"}], "from": %d}`, len(definition)-3))
	require.Equal(t, "'", inside.Objects[0].Definition)
	require.Equal(t, fmt.Sprintf("bytes %d to %d of %d", len(definition)-1, len(definition), len(definition)), inside.Objects[0].Part)
}

func TestCatalogReadsEveryOverload(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("  PERFORM pg_sleep(0);\n", 1000)
	overloads := map[string]string{
		"public.ff(integer)": "CREATE FUNCTION f(a integer) RETURNS void AS $$\n" + body + "$$",
		"public.ff(text)":    "CREATE FUNCTION f(a text) RETURNS void AS $$\n" + body + "$$",
	}
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "public",
		Functions: []*metadatapb.FunctionMetadata{
			{Name: "f", Signature: "f(integer)", Definition: overloads["public.ff(integer)"]},
			{Name: "f", Signature: "f(text)", Definition: overloads["public.ff(text)"]},
		},
		// A table can share the name of a function.
		Tables: []*metadatapb.TableMetadata{{Name: "f", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}}}},
	}}})

	first := read(t, tools, `{"objects": [{"schema": "public", "name": "f"}]}`)
	require.Len(t, first.Objects, 2, "the two overloads do not fit one result")
	require.NotNil(t, first.NextFrom)

	order, text := readAll(t, tools, `[{"schema": "public", "name": "f"}]`)
	require.Equal(t, []string{"public.f", "public.ff(integer)", "public.ff(text)"}, order)
	require.Equal(t, "TABLE public.f (\n  id bigint NOT NULL\n)", text["public.f"])
	require.Equal(t, overloads["public.ff(integer)"], text["public.ff(integer)"])
	require.Equal(t, overloads["public.ff(text)"], text["public.ff(text)"])

	matches := search(t, tools, `{"text": "a text"}`)
	require.Equal(t, []searchMatch{{Kind: kindFunction, Schema: "public", Name: "f", Signature: "f(text)", Matched: "definition"}}, matches.Matches)
}

func TestCatalogReadOfANameInManySchemas(t *testing.T) {
	t.Parallel()

	const schemaCount = 2000
	database := &metadatapb.DatabaseSchemaMetadata{}
	for i := range schemaCount {
		database.Schemas = append(database.Schemas, &metadatapb.SchemaMetadata{
			Name:   fmt.Sprintf("tenant_%04d", i),
			Tables: []*metadatapb.TableMetadata{{Name: "orders", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}}}},
		})
	}
	tools := unbounded(NewCatalogTools(storepb.Engine_SNOWFLAKE, database))

	order, text := readAll(t, tools, `[{"name": "orders"}]`)
	require.Len(t, order, schemaCount)
	require.Equal(t, "TABLE tenant_1999.orders (\n  id bigint NOT NULL\n)", text["tenant_1999.orders"])
}

func TestCatalogTellsSchemasApartByCase(t *testing.T) {
	t.Parallel()

	table := func() []*metadatapb.TableMetadata {
		return []*metadatapb.TableMetadata{{Name: "orders", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}}}}
	}
	tools := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{
		{Name: "Foo", Tables: table()},
		{Name: "foo", Tables: table()},
	}})

	// The schema is settled before the name, so a name that differs in case
	// does not reach into the other schema.
	reached := read(t, tools, `{"objects": [{"schema": "Foo", "name": "ORDERS"}]}`)
	require.Len(t, reached.Objects, 1)
	require.Equal(t, "Foo", reached.Objects[0].Schema)

	tests := []struct {
		schema string
		want   []string
	}{
		{schema: "Foo", want: []string{"Foo"}},
		{schema: "foo", want: []string{"foo"}},
		{schema: "FOO", want: []string{"Foo", "foo"}},
	}
	for _, test := range tests {
		var readSchemas, searchSchemas []string
		for _, object := range read(t, tools, fmt.Sprintf(`{"objects": [{"schema": %q, "name": "orders"}]}`, test.schema)).Objects {
			readSchemas = append(readSchemas, object.Schema)
		}
		require.Equal(t, test.want, readSchemas, test.schema)
		for _, match := range search(t, tools, fmt.Sprintf(`{"text": "orders", "schema": %q}`, test.schema)).Matches {
			searchSchemas = append(searchSchemas, match.Schema)
		}
		require.Equal(t, test.want, searchSchemas, test.schema)
	}
}

func TestCatalogAddsTheTriggersARendererLeavesOut(t *testing.T) {
	t.Parallel()

	// The trigger body also appears in the table comment, so finding the body
	// in the rendered table does not mean the renderer wrote the trigger.
	tools := NewCatalogTools(storepb.Engine_MYSQL, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Tables: []*metadatapb.TableMetadata{{
			Name:     "orders",
			Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}},
			Comment:  "SET NEW.audit_flag = 1",
			Triggers: []*metadatapb.TriggerMetadata{{Name: "flag_orders", Timing: "BEFORE", Event: "INSERT", Body: "SET NEW.audit_flag = 1"}},
		}},
	}}})
	definition := read(t, tools, `{"objects": [{"name": "orders"}]}`).Objects[0].Definition
	require.Contains(t, definition, "COMMENT='SET NEW.audit_flag = 1'")
	require.Contains(t, definition, "TRIGGER flag_orders BEFORE INSERT ON orders\nSET NEW.audit_flag = 1")

	// PostgreSQL writes the indexes of a materialized view, the fallback lists them.
	view := &metadatapb.MaterializedViewMetadata{
		Name:       "orders_daily",
		Definition: "SELECT 1 AS day",
		Indexes:    []*metadatapb.IndexMetadata{{Name: "day", Expressions: []string{"day"}, Type: "btree"}},
	}
	for engine, want := range map[storepb.Engine]string{
		storepb.Engine_POSTGRES:  "CREATE INDEX \"day\" ON \"public\".\"orders_daily\" (day);",
		storepb.Engine_SNOWFLAKE: "SELECT 1 AS day\n\nINDEX day USING btree (day)",
	} {
		tools := NewCatalogTools(engine, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:              "public",
			MaterializedViews: []*metadatapb.MaterializedViewMetadata{view},
		}}})
		definition := read(t, tools, `{"objects": [{"name": "orders_daily"}]}`).Objects[0].Definition
		require.Contains(t, definition, want)
		require.Equal(t, 1, strings.Count(definition, "(day)"), "the index is written once: %s", definition)
	}
}

func TestCatalogHoldsTheObjectsOfTheDatabase(t *testing.T) {
	t.Parallel()

	tools := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{
		Schemas: []*metadatapb.SchemaMetadata{{Name: "public"}},
		EventTriggers: []*metadatapb.EventTriggerMetadata{
			{Name: "block_drop", Event: "sql_drop", Enabled: true, Definition: "CREATE EVENT TRIGGER block_drop ON sql_drop EXECUTE FUNCTION public.refuse_drop()"},
			{Name: "log_ddl", Event: "ddl_command_end", Tags: []string{"CREATE TABLE", "ALTER TABLE"}, FunctionSchema: "public", FunctionName: "log_ddl"},
		},
		Extensions: []*metadatapb.ExtensionMetadata{{Name: "pg_partman", Schema: "partman", Version: "5.0.1", Description: "Manage partitioned tables by time or ID"}},
	})

	require.Equal(t, []searchMatch{
		{Kind: kindEventTrigger, Name: "block_drop", Matched: "definition"},
		{Kind: kindEventTrigger, Name: "log_ddl", Matched: "definition"},
	}, search(t, tools, `{"text": "EXECUTE FUNCTION"}`).Matches)

	result := read(t, tools, `{"objects": [{"name": "block_drop"}, {"name": "log_ddl"}, {"schema": "partman", "name": "pg_partman"}]}`)
	require.Len(t, result.Objects, 3)
	require.Equal(t, "CREATE EVENT TRIGGER block_drop ON sql_drop EXECUTE FUNCTION public.refuse_drop()", result.Objects[0].Definition)
	require.Equal(t, "EVENT TRIGGER log_ddl ON ddl_command_end WHEN TAG IN ('CREATE TABLE', 'ALTER TABLE') EXECUTE FUNCTION public.log_ddl()\nDISABLED", result.Objects[1].Definition)
	require.Equal(t, kindExtension, result.Objects[2].Kind)
	require.Equal(t, "EXTENSION partman.pg_partman VERSION 5.0.1\n\nCOMMENT: Manage partitioned tables by time or ID", result.Objects[2].Definition)
}

func TestCatalogSearchFoldsCase(t *testing.T) {
	t.Parallel()

	// The name ends in a capital sigma. Its lower case is σ, and the search
	// text ends in ς, the lower case sigma of a word's end.
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:  "PUBLIC",
		Views: []*metadatapb.ViewMetadata{{Name: "ΟΣ", Definition: "SELECT ΤΙΜΉ FROM ORDERS"}},
	}}})
	for text, matched := range map[string]string{"ος": "name", "τιμή": "definition", "from orders": "definition"} {
		require.Equal(t, []searchMatch{{Kind: kindView, Schema: "PUBLIC", Name: "ΟΣ", Matched: matched}}, search(t, tools, fmt.Sprintf(`{"text": %q}`, text)).Matches, text)
	}
}

func TestCatalogRendersOnlyWhatItReads(t *testing.T) {
	t.Parallel()

	rendered := func(tools *catalogTools) []string {
		var names []string
		for _, object := range tools.objects {
			if object.definition != nil {
				names = append(names, object.schema+"."+object.name)
			}
		}
		return names
	}

	tools := pgShop()
	read(t, tools, `{"objects": [{"schema": "public", "name": "customers"}]}`)
	require.Equal(t, []string{"public.customers"}, rendered(tools))

	// A match on the name needs no definition.
	search(t, tools, `{"text": "", "schema": "archive"}`)
	require.Equal(t, []string{"public.customers"}, rendered(tools))

	search(t, tools, `{"text": "legacy_status"}`)
	require.Len(t, rendered(tools), 7)
}

func TestCutText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		text   string
		begin  int
		budget int
		want   string
	}{
		{name: "the rest fits", text: "a\nbb\nccc", begin: 5, budget: 5, want: "ccc"},
		{name: "behind the last line break", text: "a\nbb\nccc", begin: 0, budget: 6, want: "a\nbb\n"},
		{name: "inside a line without a break", text: "aaaaaaaa", begin: 2, budget: 3, want: "aaa"},
		{name: "never inside a character", text: "数数数", begin: 0, budget: 4, want: "数"},
		{name: "a budget under one character takes it whole", text: "数数数", begin: 3, budget: 2, want: "数"},
		{name: "bytes that are not text are cut at the budget", text: strings.Repeat("\x80", 20), begin: 0, budget: 10, want: strings.Repeat("\x80", 6)},
	}
	for _, test := range tests {
		stop := cutText(test.text, test.begin, test.budget)
		require.Equal(t, test.want, test.text[test.begin:stop], test.name)
	}
}

func TestCatalogExplainsBadArguments(t *testing.T) {
	t.Parallel()

	var tooMany readArguments
	for i := range maxReadObjects + 1 {
		tooMany.Objects = append(tooMany.Objects, objectName{Name: fmt.Sprintf("table_%d", i)})
	}
	tooManyArguments, err := jsonv2.Marshal(tooMany)
	require.NoError(t, err)

	tests := []struct {
		name      string
		tool      string
		arguments string
		want      string
	}{
		{name: "search arguments that are not JSON", tool: toolSearch, arguments: `orders`, want: "invalid arguments"},
		{name: "search from a string", tool: toolSearch, arguments: `{"text": "orders", "from": "ten"}`, want: "invalid arguments"},
		{name: "search from below zero", tool: toolSearch, arguments: `{"text": "orders", "from": -1}`, want: "from must be 0 or greater"},
		{name: "read arguments that are not JSON", tool: toolRead, arguments: `orders`, want: "invalid arguments"},
		{name: "read objects as names", tool: toolRead, arguments: `{"objects": ["orders"]}`, want: "invalid arguments"},
		{name: "read nothing", tool: toolRead, arguments: `{"objects": []}`, want: "pass at least one object"},
		{name: "read too many", tool: toolRead, arguments: string(tooManyArguments), want: "read takes at most 20 names in one call and you passed 21"},
		{name: "read from below zero", tool: toolRead, arguments: `{"objects": [{"name": "orders"}], "from": -1}`, want: "from must be 0 or greater"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// The model can correct its arguments, so they are not a failure of the review.
			output, err := pgShop().Call(context.Background(), test.tool, test.arguments)
			require.NoError(t, err)
			require.Contains(t, output, test.want)
		})
	}

	_, err = pgShop().Call(context.Background(), "drop", `{}`)
	require.ErrorContains(t, err, `tool "drop" does not exist`)
}

func TestCatalogOfAnEmptyDatabase(t *testing.T) {
	t.Parallel()

	for _, schema := range []*metadatapb.DatabaseSchemaMetadata{nil, {}} {
		tools := NewCatalogTools(storepb.Engine_POSTGRES, schema)
		require.Empty(t, search(t, tools, `{"text": ""}`).Matches)
		result := read(t, tools, `{"objects": [{"name": "orders"}]}`)
		require.Empty(t, result.Objects)
		require.Equal(t, []objectName{{Name: "orders"}}, result.NotFound)
	}
}

func TestCatalogReadWritesOnlyTheSequenceAttributesSynced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		engine   storepb.Engine
		schema   string
		sequence *metadatapb.SequenceMetadata
		want     string
	}{
		{
			name:     "a sync that holds no attribute",
			engine:   storepb.Engine_ORACLE,
			schema:   "SHOP",
			sequence: &metadatapb.SequenceMetadata{Name: "ORDER_SEQ"},
			want:     "SEQUENCE SHOP.ORDER_SEQ\n-- not synced: start, increment",
		},
		{
			name:     "a sync that holds some",
			engine:   storepb.Engine_ORACLE,
			schema:   "SHOP",
			sequence: &metadatapb.SequenceMetadata{Name: "INVOICE_SEQ", Increment: "10"},
			want:     "SEQUENCE SHOP.INVOICE_SEQ INCREMENT 10\n-- not synced: start",
		},
		{
			name:     "a sync that holds bounds and cycling",
			engine:   storepb.Engine_TIDB,
			sequence: &metadatapb.SequenceMetadata{Name: "ticket_seq", Start: "1", Increment: "1", MinValue: "1", MaxValue: "1000", CacheSize: "20", Cycle: true},
			want:     "SEQUENCE ticket_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 1000 CACHE 20 CYCLE",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tools := NewCatalogTools(tc.engine, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
				Name:      tc.schema,
				Sequences: []*metadatapb.SequenceMetadata{tc.sequence},
			}}})
			_, text := readAll(t, tools, fmt.Sprintf(`[{"name": %q}]`, tc.sequence.GetName()))
			require.Equal(t, map[string]string{tc.schema + "." + tc.sequence.GetName(): tc.want}, text)
		})
	}
}

func TestCatalogReadWritesTheTableFactsSyncedWithoutARenderer(t *testing.T) {
	t.Parallel()

	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Tables: []*metadatapb.TableMetadata{{
			Name:      "orders",
			Engine:    "Aria",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
			Columns: []*metadatapb.ColumnMetadata{
				{Name: "id", Type: "bigint", IsIdentity: true},
				{Name: "note", Type: "varchar(64)", Nullable: true, CharacterSet: "utf8mb4", Collation: "utf8mb4_bin"},
				{Name: "total_cents", Type: "bigint", Generation: &metadatapb.GenerationMetadata{Type: metadatapb.GenerationMetadata_TYPE_STORED, Expression: "total * 100"}},
				{Name: "updated_at", Type: "timestamp", Default: "CURRENT_TIMESTAMP", OnUpdate: "CURRENT_TIMESTAMP"},
				{Name: "shard", Type: "int", Nullable: true, IsInvisible: true},
			},
			Indexes: []*metadatapb.IndexMetadata{
				{Name: "idx_orders_recent", Expressions: []string{"updated_at", "id"}, Descending: []bool{true, false}, Type: "BTREE"},
				{Name: "idx_orders_note", Expressions: []string{"note"}, KeyLength: []int64{10}, Type: "BTREE"},
			},
			ForeignKeys: []*metadatapb.ForeignKeyMetadata{{
				Name: "orders_customer_fk", Columns: []string{"customer_id"},
				ReferencedTable: "customers", ReferencedColumns: []string{"id"},
				MatchType: "SIMPLE", OnDelete: "CASCADE", OnUpdate: "RESTRICT",
			}},
		}},
	}}})

	_, text := readAll(t, tools, `[{"name": "orders"}]`)
	require.Equal(t, strings.Join([]string{
		"TABLE orders (",
		"  id bigint IDENTITY NOT NULL",
		"  note varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin",
		"  total_cents bigint GENERATED ALWAYS AS (total * 100) STORED NOT NULL",
		"  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP",
		"  shard int INVISIBLE",
		") ENGINE=Aria DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
		"",
		"INDEX idx_orders_recent USING BTREE (updated_at DESC, id)",
		"INDEX idx_orders_note USING BTREE (note(10))",
		"",
		"FOREIGN KEY orders_customer_fk (customer_id) REFERENCES customers (id) MATCH SIMPLE ON DELETE CASCADE ON UPDATE RESTRICT",
	}, "\n"), text[".orders"])
}

func TestCatalogReadsAMariaDBTableAsItsSyncStoresIt(t *testing.T) {
	t.Parallel()

	// The shape the MySQL-family sync gives a MariaDB table: AUTO_INCREMENT
	// and a NULL default kept in the default, FULLTEXT and SPATIAL as the
	// index type, -1 for a part with no prefix, and NONE for the match type.
	tools := NewCatalogTools(storepb.Engine_MARIADB, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Tables: []*metadatapb.TableMetadata{{
			Name:      "orders",
			Engine:    "InnoDB",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_general_ci",
			Columns: []*metadatapb.ColumnMetadata{
				{Name: "id", Type: "bigint", Default: "AUTO_INCREMENT"},
				{Name: "note", Type: "text", Nullable: true, Default: "NULL"},
				{Name: "spot", Type: "point"},
				{Name: "customer_id", Type: "bigint"},
			},
			Indexes: []*metadatapb.IndexMetadata{
				{Name: "PRIMARY", Expressions: []string{"id"}, KeyLength: []int64{-1}, Type: "BTREE", Unique: true, Primary: true, Visible: true},
				{Name: "ft_note", Expressions: []string{"note"}, KeyLength: []int64{-1}, Type: "FULLTEXT", Visible: true},
				{Name: "sp_spot", Expressions: []string{"spot"}, KeyLength: []int64{32}, Type: "SPATIAL", Visible: true},
				{Name: "idx_note", Expressions: []string{"note"}, KeyLength: []int64{10}, Type: "BTREE", Visible: true},
			},
			ForeignKeys: []*metadatapb.ForeignKeyMetadata{{
				Name: "orders_customer_fk", Columns: []string{"customer_id"},
				ReferencedTable: "customers", ReferencedColumns: []string{"id"},
				OnDelete: "CASCADE", OnUpdate: "RESTRICT", MatchType: "NONE",
			}},
			Triggers: []*metadatapb.TriggerMetadata{{Name: "stamp", Timing: "BEFORE", Event: "INSERT", Body: "SET NEW.note = 'new'", SqlMode: "PIPES_AS_CONCAT"}},
		}},
		Functions: []*metadatapb.FunctionMetadata{{
			Name:              "order_total",
			Definition:        "CREATE FUNCTION order_total(id bigint) RETURNS bigint RETURN id || 0",
			SqlMode:           "PIPES_AS_CONCAT",
			DatabaseCollation: "utf8mb4_general_ci",
		}},
	}}})

	_, text := readAll(t, tools, `[{"name": "orders"}, {"name": "order_total"}]`)
	require.Equal(t, strings.Join([]string{
		"TABLE orders (",
		"  id bigint NOT NULL AUTO_INCREMENT",
		"  note text",
		"  spot point NOT NULL",
		"  customer_id bigint NOT NULL",
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci",
		"",
		"PRIMARY KEY PRIMARY USING BTREE (id)",
		"FULLTEXT INDEX ft_note (note)",
		"SPATIAL INDEX sp_spot (spot)",
		"INDEX idx_note USING BTREE (note(10))",
		"",
		"FOREIGN KEY orders_customer_fk (customer_id) REFERENCES customers (id) ON DELETE CASCADE ON UPDATE RESTRICT",
		"",
		"TRIGGER stamp BEFORE INSERT ON orders",
		"SET NEW.note = 'new'",
		"",
		"-- created with sql_mode=PIPES_AS_CONCAT",
	}, "\n"), text[".orders"])
	require.Equal(t, "CREATE FUNCTION order_total(id bigint) RETURNS bigint RETURN id || 0\n\n-- created with sql_mode=PIPES_AS_CONCAT\n\n-- created with database_collation=utf8mb4_general_ci", text[".order_total"])
}

func TestCatalogReadSkipsWhatASyncLeavesEmpty(t *testing.T) {
	t.Parallel()

	// DynamoDB syncs an empty range index for a table with no range key.
	tools := NewCatalogTools(storepb.Engine_DYNAMODB, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Tables: []*metadatapb.TableMetadata{{
			Name:    "sessions",
			Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "S"}},
			Indexes: []*metadatapb.IndexMetadata{{Name: "id", Expressions: []string{"id"}, Type: "HASH"}, {Type: "RANGE"}},
		}},
	}}})

	_, text := readAll(t, tools, `[{"name": "sessions"}]`)
	require.Equal(t, "TABLE sessions (\n  id S\n)\n\nINDEX id USING HASH (id)", text[".sessions"])
}

func TestCatalogReadNamesTheDisabledRules(t *testing.T) {
	t.Parallel()

	rules := func() []*metadatapb.RuleMetadata {
		return []*metadatapb.RuleMetadata{
			{Name: "keep", Event: "DELETE", Action: "NOTHING", IsInstead: true, IsEnabled: true, Definition: "CREATE RULE keep AS ON DELETE TO public.t DO INSTEAD NOTHING"},
			{Name: "paused", Event: "UPDATE", Action: "NOTHING", IsInstead: true, Definition: "CREATE RULE paused AS ON UPDATE TO public.t DO INSTEAD NOTHING"},
		}
	}
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_SNOWFLAKE} {
		tools := NewCatalogTools(engine, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{{Name: "t", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}, Rules: rules()}},
			Views:  []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT 1 AS one", Rules: rules()}},
		}}})
		_, text := readAll(t, tools, `[{"name": "t"}, {"name": "v"}]`)
		for _, name := range []string{"public.t", "public.v"} {
			require.Equal(t, 1, strings.Count(text[name], "RULE paused DISABLED"), "%s on %s:\n%s", name, engine, text[name])
			require.NotContains(t, text[name], "RULE keep DISABLED", "%s on %s", name, engine)
		}
	}
}

func TestCatalogReadClaimsNothingAHiveSyncDoesNotRead(t *testing.T) {
	t.Parallel()

	tools := NewCatalogTools(storepb.Engine_HIVE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Tables: []*metadatapb.TableMetadata{{
			Name:       "events",
			Columns:    []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}},
			Partitions: []*metadatapb.TablePartitionMetadata{{Name: "day=2026-10-15"}},
		}},
		ExternalTables: []*metadatapb.ExternalTableMetadata{{
			Name:    "raw_events",
			Columns: []*metadatapb.ColumnMetadata{{Name: "payload", Type: "string"}},
		}},
	}}})

	_, text := readAll(t, tools, `[{"name": "events"}, {"name": "raw_events"}]`)
	require.Equal(t, "TABLE events (\n  id bigint\n)\n\nPARTITION \"day=2026-10-15\"", text[".events"])
	require.Equal(t, "EXTERNAL TABLE raw_events (\n  payload string\n)", text[".raw_events"])
}

func TestCatalogReadReadsHowEachSyncStoresAFact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		engine storepb.Engine
		table  *metadatapb.TableMetadata
		want   string
	}{
		{
			name:   "AUTO_INCREMENT kept in the default",
			engine: storepb.Engine_STARROCKS,
			table:  &metadatapb.TableMetadata{Name: "t", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint", Default: "AUTO_INCREMENT"}}},
			want:   "TABLE t (\n  id bigint NOT NULL AUTO_INCREMENT\n)",
		},
		{
			name:   "NULL stored for a column without a default",
			engine: storepb.Engine_BIGQUERY,
			table:  &metadatapb.TableMetadata{Name: "t", Columns: []*metadatapb.ColumnMetadata{{Name: "note", Type: "STRING", Nullable: true, Default: "NULL"}}},
			want:   "TABLE t (\n  note STRING\n)",
		},
		{
			name:   "a storage engine Hive never reads",
			engine: storepb.Engine_HIVE,
			table:  &metadatapb.TableMetadata{Name: "t", Engine: "HDFS", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}}},
			want:   "TABLE t (\n  id bigint\n)",
		},
		{
			name:   "index options kept as the definition",
			engine: storepb.Engine_CASSANDRA,
			table: &metadatapb.TableMetadata{
				Name:    "t",
				Columns: []*metadatapb.ColumnMetadata{{Name: "email", Type: "text", Nullable: true}},
				Indexes: []*metadatapb.IndexMetadata{{Name: "t_email_idx", Type: "COMPOSITES", Definition: `{"target": "email"}`}},
			},
			want: "TABLE t (\n  email text\n)\n\nINDEX t_email_idx USING COMPOSITES OPTIONS {\"target\": \"email\"}",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tools := NewCatalogTools(tc.engine, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{tc.table},
			}}})
			_, text := readAll(t, tools, `[{"name": "t"}]`)
			require.Equal(t, tc.want, text[".t"])
		})
	}
}

func TestCatalogReadsTheSchemaTaskAndAttributeFactsSynced(t *testing.T) {
	t.Parallel()

	tools := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{
		{
			Name:    "billing",
			Owner:   "billing_owner",
			Comment: "invoices and payments",
			CompositeTypes: []*metadatapb.CompositeTypeMetadata{{
				Name:       "address",
				Attributes: []*metadatapb.CompositeTypeAttribute{{Name: "city", Type: "text", Collation: `"C"`}, {Name: "zip", Type: "text"}},
			}},
		},
		{Name: "plain"},
	}})

	_, text := readAll(t, tools, `[{"name": "billing"}, {"name": "address"}]`)
	require.Equal(t, "SCHEMA billing OWNER billing_owner\n\nCOMMENT: invoices and payments", text[".billing"])
	require.Contains(t, text["billing.address"], `city text COLLATE "C", zip text`)

	matches := search(t, tools, `{"text": "invoices"}`).Matches
	require.Equal(t, []searchMatch{{Kind: kindSchema, Name: "billing", Matched: "definition"}}, matches)
	require.Empty(t, read(t, tools, `{"objects": [{"name": "plain"}]}`).Objects, "a schema with no owner or comment is not an object")
	inBilling := search(t, tools, `{"text": "", "schema": "billing"}`).Matches
	require.Equal(t, []string{kindSchema, kindCompositeType}, []string{inBilling[0].Kind, inBilling[1].Kind})
	require.Len(t, read(t, tools, `{"objects": [{"schema": "billing", "name": "billing"}]}`).Objects, 1)

	tasks := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "PUBLIC",
		Tasks: []*metadatapb.TaskMetadata{
			{Name: "refresh", Definition: "CREATE TASK refresh SCHEDULE = '5 MINUTE' AS CALL refresh()", State: metadatapb.TaskMetadata_STATE_SUSPENDED},
			{Name: "unknown", Definition: "CREATE TASK unknown AS SELECT 1"},
		},
	}}})
	_, text = readAll(t, tasks, `[{"name": "refresh"}, {"name": "unknown"}]`)
	require.Equal(t, "CREATE TASK refresh SCHEDULE = '5 MINUTE' AS CALL refresh()\n\nSTATE: suspended", text["PUBLIC.refresh"])
	require.Equal(t, "CREATE TASK unknown AS SELECT 1", text["PUBLIC.unknown"])
}

func TestCatalogReadsTheFactsOutsideTheDefinition(t *testing.T) {
	t.Parallel()

	owned := func() *metadatapb.TableMetadata {
		return &metadatapb.TableMetadata{
			Name:           "orders",
			Owner:          "app_owner",
			SortingKeys:    []string{"created_at", "id"},
			ShardingInfo:   "SHARD_ROW_ID_BITS=4",
			PrimaryKeyType: "CLUSTERED",
			Columns:        []*metadatapb.ColumnMetadata{{Name: "id", Type: "bigint"}},
		}
	}
	// A renderer that writes the owner writes it once; otherwise it is a line of its own.
	postgres := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{Name: "public", Tables: []*metadatapb.TableMetadata{owned()}}}})
	_, text := readAll(t, postgres, `[{"name": "orders"}]`)
	require.Equal(t, 1, strings.Count(text["public.orders"], "app_owner"), text["public.orders"])

	snowflake := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:   "PUBLIC",
		Tables: []*metadatapb.TableMetadata{owned()},
		Streams: []*metadatapb.StreamMetadata{
			{Name: "order_changes", Definition: "CREATE STREAM order_changes ON TABLE orders", Owner: "etl", Stale: true},
			{Name: "fresh_changes", Definition: "CREATE STREAM fresh_changes ON TABLE orders"},
		},
		Tasks:          []*metadatapb.TaskMetadata{{Name: "refresh", Definition: "CREATE TASK refresh AS SELECT 1", Owner: "etl"}},
		ExternalTables: []*metadatapb.ExternalTableMetadata{{Name: "raw", ExternalServerName: "lake", ExternalDatabaseName: "landing", Columns: []*metadatapb.ColumnMetadata{{Name: "payload", Type: "variant", Nullable: true}}}},
	}}})
	_, text = readAll(t, snowflake, `[{"name": "orders"}, {"name": "order_changes"}, {"name": "fresh_changes"}, {"name": "refresh"}, {"name": "raw"}]`)
	for _, want := range []string{"OWNER: app_owner", "SORTING KEY: created_at, id", "SHARDING: SHARD_ROW_ID_BITS=4", "PRIMARY KEY TYPE: CLUSTERED"} {
		require.Contains(t, text["PUBLIC.orders"], want)
	}
	require.Equal(t, "CREATE STREAM order_changes ON TABLE orders\n\nOWNER: etl\n\nSTALE: the stream can no longer return the changes since its offset", text["PUBLIC.order_changes"])
	require.Equal(t, "CREATE STREAM fresh_changes ON TABLE orders", text["PUBLIC.fresh_changes"])
	require.Equal(t, "CREATE TASK refresh AS SELECT 1\n\nOWNER: etl", text["PUBLIC.refresh"])
	require.Equal(t, "EXTERNAL TABLE PUBLIC.raw ON SERVER lake DATABASE landing (\n  payload variant\n)", text["PUBLIC.raw"])

	mysql := NewCatalogTools(storepb.Engine_MYSQL, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Events: []*metadatapb.EventMetadata{{
			Name:       "purge",
			Definition: "CREATE EVENT purge ON SCHEDULE EVERY 1 DAY STARTS '2026-10-15 00:00:00' DO DELETE FROM logs",
			TimeZone:   "+08:00",
			SqlMode:    "ANSI_QUOTES",
		}},
	}}})
	_, text = readAll(t, mysql, `[{"name": "purge"}]`)
	require.Equal(t, "CREATE EVENT purge ON SCHEDULE EVERY 1 DAY STARTS '2026-10-15 00:00:00' DO DELETE FROM logs\n\n-- created with sql_mode=ANSI_QUOTES, time_zone=+08:00", text[".purge"])
}

func TestCatalogReadsTheFactsARendererLeavesOut(t *testing.T) {
	t.Parallel()

	read := func(engine storepb.Engine, schema *metadatapb.SchemaMetadata, name string) string {
		t.Helper()
		tools := NewCatalogTools(engine, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{schema}})
		_, text := readAll(t, tools, fmt.Sprintf(`[{"name": %q}]`, name))
		require.Len(t, text, 1)
		for _, definition := range text {
			return definition
		}
		return ""
	}

	// MySQL's routine renderer writes the session settings but not the database collation.
	routine := read(storepb.Engine_MYSQL, &metadatapb.SchemaMetadata{Functions: []*metadatapb.FunctionMetadata{{
		Name: "total", Definition: "CREATE FUNCTION total() RETURNS int RETURN 1", SqlMode: "ANSI_QUOTES", DatabaseCollation: "utf8mb4_bin",
	}}}, "total")
	require.Equal(t, 1, strings.Count(routine, "utf8mb4_bin"), routine)
	require.Equal(t, 1, strings.Count(routine, "ANSI_QUOTES"), routine)

	// An invisible index is named where the sync reads visibility, and never elsewhere.
	indexed := func() *metadatapb.SchemaMetadata {
		return &metadatapb.SchemaMetadata{Name: "APP", Tables: []*metadatapb.TableMetadata{{
			Name:    "ORDERS",
			Columns: []*metadatapb.ColumnMetadata{{Name: "ID", Type: "NUMBER"}},
			Indexes: []*metadatapb.IndexMetadata{
				{Name: "ORDERS_ID", Expressions: []string{"ID"}, Type: "NORMAL", Visible: true},
				{Name: "ORDERS_HIDDEN", Expressions: []string{"ID"}, Type: "NORMAL"},
			},
		}}}
	}
	oracle := read(storepb.Engine_ORACLE, indexed(), "ORDERS")
	require.Contains(t, oracle, "CREATE TABLE", "rendered by the Oracle renderer")
	require.Equal(t, 1, strings.Count(oracle, "INDEX ORDERS_HIDDEN INVISIBLE"), oracle)
	require.NotContains(t, oracle, "INDEX ORDERS_ID INVISIBLE")
	require.NotContains(t, read(storepb.Engine_POSTGRES, indexed(), "ORDERS"), "INVISIBLE")

	// SQL Server holds a column default in a constraint a change drops by name.
	mssql := read(storepb.Engine_MSSQL, &metadatapb.SchemaMetadata{Name: "dbo", Tables: []*metadatapb.TableMetadata{{
		Name:    "orders",
		Columns: []*metadatapb.ColumnMetadata{{Name: "status", Type: "int", Default: "((0))", DefaultConstraintName: "DF_orders_status"}},
	}}}, "orders")
	require.Contains(t, mssql, "CREATE TABLE", "rendered by the SQL Server renderer")
	require.Equal(t, 1, strings.Count(mssql, "DF_orders_status"), mssql)

	// ClickHouse's renderer leaves data-skipping indexes out, and they come back with their granularity.
	clickhouse := read(storepb.Engine_CLICKHOUSE, &metadatapb.SchemaMetadata{Tables: []*metadatapb.TableMetadata{{
		Name:    "events",
		Columns: []*metadatapb.ColumnMetadata{{Name: "url", Type: "String"}},
		Indexes: []*metadatapb.IndexMetadata{{Name: "url_bloom", Expressions: []string{"url"}, Type: "bloom_filter", Granularity: 4}},
	}}}, "events")
	require.Contains(t, clickhouse, "CREATE TABLE", "rendered by the ClickHouse renderer")
	require.Contains(t, clickhouse, "INDEX url_bloom USING bloom_filter (url) GRANULARITY 4")
}

func TestCatalogStopsWhenTheReviewEnds(t *testing.T) {
	t.Parallel()

	tools := ordersCatalog()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := tools.Call(ctx, toolSearch, `{"text": "legacy_status"}`)
	require.ErrorIs(t, err, context.Canceled)
	_, err = tools.Call(ctx, toolRead, `{"objects": [{"name": "orders"}]}`)
	require.ErrorIs(t, err, context.Canceled)

	// The canceled search kept nothing, so the next one is whole.
	result := search(t, tools, `{"text": "legacy_status"}`)
	require.Equal(t, 2, result.Total)
	// A search answered from the cache stops too.
	_, err = tools.Call(ctx, toolSearch, `{"text": "legacy_status"}`)
	require.ErrorIs(t, err, context.Canceled)
}

func TestCatalogReadsTheSequencePositionAtSync(t *testing.T) {
	t.Parallel()

	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_ORACLE} {
		tools := NewCatalogTools(engine, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:      "app",
			Sequences: []*metadatapb.SequenceMetadata{{Name: "order_id_seq", DataType: "integer", Start: "1", Increment: "1", MaxValue: "2147483647", LastValue: "2147483000"}},
		}}})
		_, text := readAll(t, tools, `[{"name": "order_id_seq"}]`)
		require.Equal(t, 1, strings.Count(text["app.order_id_seq"], "LAST VALUE AT SYNC: 2147483000"), "%s: %s", engine, text["app.order_id_seq"])
	}
}

func statistics(rows int64, data int64, index int64) *tableStatistics {
	return &tableStatistics{Rows: &rows, DataBytes: &data, IndexBytes: &index}
}

func TestCatalogShowsOnlyTheStatisticsTheSyncReads(t *testing.T) {
	t.Parallel()

	table := func(rows int64) *metadatapb.TableMetadata {
		return &metadatapb.TableMetadata{Name: "t", RowCount: rows, Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}}}
	}
	statisticsOn := func(engine storepb.Engine, rows int64) *tableStatistics {
		t.Helper()
		tools := NewCatalogTools(engine, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{Name: "s", Tables: []*metadatapb.TableMetadata{table(rows)}}}})
		return read(t, tools, `{"objects": [{"name": "t"}]}`).Objects[0].Statistics
	}
	rows := int64(100)
	// SQL Server syncs the row count alone, so its sizes are not shown as 0.
	require.Equal(t, &tableStatistics{Rows: &rows}, statisticsOn(storepb.Engine_MSSQL, 100))
	// An engine whose sync reads no statistics shows a value only when it is not 0.
	require.Nil(t, statisticsOn(storepb.Engine_CASSANDRA, 0))
	require.Equal(t, &tableStatistics{Rows: &rows}, statisticsOn(storepb.Engine_CASSANDRA, 100))
	// Hive syncs the row count and the data size.
	zero := int64(0)
	require.Equal(t, &tableStatistics{Rows: &zero, DataBytes: &zero}, statisticsOn(storepb.Engine_HIVE, 0))
	// Redshift writes 0 for the sizes, and a datashare database reads nothing.
	require.Equal(t, &tableStatistics{Rows: &zero}, statisticsOn(storepb.Engine_REDSHIFT, 0))
	datashare := NewCatalogTools(storepb.Engine_REDSHIFT, &metadatapb.DatabaseSchemaMetadata{Datashare: true, Schemas: []*metadatapb.SchemaMetadata{{Name: "s", Tables: []*metadatapb.TableMetadata{table(0)}}}})
	require.Nil(t, read(t, datashare, `{"objects": [{"name": "t"}]}`).Objects[0].Statistics)
}

func TestCatalogMarksTheObjectsAnExtensionOwns(t *testing.T) {
	t.Parallel()

	tools := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "public",
		Tables: []*metadatapb.TableMetadata{
			{Name: "spatial_ref_sys", SkipDump: true, Columns: []*metadatapb.ColumnMetadata{{Name: "srid", Type: "integer"}}},
			{Name: "orders", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}}},
		},
		Views:     []*metadatapb.ViewMetadata{{Name: "geometry_columns", Definition: "SELECT 1", SkipDump: true}},
		Functions: []*metadatapb.FunctionMetadata{{Name: "postgis_version", Signature: "postgis_version()", Definition: "CREATE FUNCTION public.postgis_version() RETURNS text LANGUAGE c AS '$libdir/postgis'", SkipDump: true}},
	}}})
	_, text := readAll(t, tools, `[{"name": "spatial_ref_sys"}, {"name": "orders"}, {"name": "geometry_columns"}, {"name": "postgis_version"}]`)
	const mark = "MEMBER OF AN EXTENSION"
	for _, name := range []string{"public.spatial_ref_sys", "public.geometry_columns", "public.postgis_versionpostgis_version()"} {
		require.Equal(t, 1, strings.Count(text[name], mark), "%s:\n%s", name, text[name])
	}
	require.NotContains(t, text["public.orders"], mark)

	// The database-level and table-level objects an extension owns are marked too.
	database := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{
		EventTriggers: []*metadatapb.EventTriggerMetadata{{Name: "ddl_guard", Event: "ddl_command_end", FunctionSchema: "public", FunctionName: "guard", Enabled: true, SkipDump: true}},
		Schemas: []*metadatapb.SchemaMetadata{{
			Name:     "topology",
			SkipDump: true,
			Tables: []*metadatapb.TableMetadata{{
				Name:     "layer",
				Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}},
				Triggers: []*metadatapb.TriggerMetadata{{Name: "layer_integrity", Body: "CREATE TRIGGER layer_integrity BEFORE UPDATE ON topology.layer FOR EACH ROW EXECUTE FUNCTION topology.check()", SkipDump: true}},
			}},
		}},
	})
	_, text = readAll(t, database, `[{"name": "ddl_guard"}, {"name": "topology"}, {"name": "layer"}]`)
	for _, name := range []string{".ddl_guard", ".topology", "topology.layer"} {
		require.Equal(t, 1, strings.Count(text[name], mark), "%s:\n%s", name, text[name])
	}
}

func TestCatalogKeepsEachEnumLabelWhole(t *testing.T) {
	t.Parallel()

	tools := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:      "public",
		EnumTypes: []*metadatapb.EnumTypeMetadata{{Name: "state", Values: []string{"pending,review", "done", "it's done"}}},
	}}})
	_, text := readAll(t, tools, `[{"name": "state"}]`)
	require.Equal(t, "ENUM TYPE public.state: 'pending,review', 'done', 'it''s done'", text["public.state"])
}

func TestCatalogKeepsEachNameInAListWhole(t *testing.T) {
	t.Parallel()

	require.Equal(t, "order_id", ident("order_id"))
	require.Equal(t, "订单", ident("订单"))
	require.Equal(t, `"a, b"`, ident("a, b"))
	require.Equal(t, `"say ""hi"""`, ident(`say "hi"`))
	require.Equal(t, `"2fa"`, ident("2fa"))

	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Tables: []*metadatapb.TableMetadata{{
			Name:    "t",
			Columns: []*metadatapb.ColumnMetadata{{Name: "a, b", Type: "int"}},
			ForeignKeys: []*metadatapb.ForeignKeyMetadata{{
				Name: "fk", Columns: []string{"a, b"}, ReferencedTable: "u", ReferencedColumns: []string{"id", "x, y"},
			}},
		}},
	}}})
	_, text := readAll(t, tools, `[{"name": "t"}]`)
	require.Contains(t, text[".t"], "TABLE t (\n  \"a, b\" int NOT NULL\n)")
	require.Contains(t, text[".t"], `FOREIGN KEY fk ("a, b") REFERENCES u (id, "x, y")`)
}

func TestCatalogQuotesAnIndexedColumnButNotAnExpression(t *testing.T) {
	t.Parallel()

	tools := NewCatalogTools(storepb.Engine_MARIADB, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Tables: []*metadatapb.TableMetadata{{
			Name:    "t",
			Columns: []*metadatapb.ColumnMetadata{{Name: "a, b", Type: "int"}, {Name: "email", Type: "varchar(64)"}},
			Indexes: []*metadatapb.IndexMetadata{
				{Name: "by_ab", Expressions: []string{"a, b"}, KeyLength: []int64{-1}, Type: "BTREE", Visible: true},
				{Name: "by_email", Expressions: []string{"lower(email)"}, KeyLength: []int64{-1}, Type: "BTREE", Visible: true},
			},
		}},
	}}})
	_, text := readAll(t, tools, `[{"name": "t"}]`)
	require.Contains(t, text[".t"], `INDEX by_ab USING BTREE ("a, b")`)
	require.Contains(t, text[".t"], "INDEX by_email USING BTREE (lower(email))")
}

func TestCatalogListsTheColumnsOfAViewWithAHiddenDefinition(t *testing.T) {
	t.Parallel()

	columns := func() []*metadatapb.ColumnMetadata {
		return []*metadatapb.ColumnMetadata{{Name: "order_id", Type: "int"}, {Name: "total", Type: "decimal(10,2)"}}
	}
	for engine, definition := range map[storepb.Engine]string{
		storepb.Engine_MSSQL:     "-- The definition of this view is encrypted.",
		storepb.Engine_SNOWFLAKE: "",
	} {
		tools := NewCatalogTools(engine, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:  "app",
			Views: []*metadatapb.ViewMetadata{{Name: "order_totals", Definition: definition, Columns: columns()}},
		}}})
		_, text := readAll(t, tools, `[{"name": "order_totals"}]`)
		require.Equal(t, 1, strings.Count(text["app.order_totals"], "COLUMNS:\n  order_id int\n  total decimal(10,2)"), "%s:\n%s", engine, text["app.order_totals"])
	}
}

// unbounded lifts the tool output budget of a review, for a test of paging
// that reads more than one review may.
func unbounded(tools Tools) Tools {
	catalog, ok := tools.(*catalogTools)
	if !ok {
		panic("not the catalog tools")
	}
	catalog.budget = math.MaxInt
	return catalog
}

func TestCatalogBoundsTheOutputOfAWholeReview(t *testing.T) {
	t.Parallel()

	// Definitions with nothing to escape, so the output measures exactly.
	large := strings.Repeat("x", maxToolResultBytes)
	var functions []*metadatapb.FunctionMetadata
	for i := range 8 {
		functions = append(functions, &metadatapb.FunctionMetadata{Name: fmt.Sprintf("f%d", i), Definition: large})
	}
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{Name: "PUBLIC", Functions: functions}}})

	total, notes := 0, 0
	for i := range 8 {
		output, err := tools.Call(context.Background(), toolRead, fmt.Sprintf(`{"objects": [{"name": "f%d"}]}`, i))
		require.NoError(t, err)
		total += len(output)
		if output == "budget spent" || strings.HasPrefix(output, "the catalog budget of this review is spent") {
			notes += len(output)
		}
	}
	require.Positive(t, notes, "the budget runs out before the eighth read")
	// Each call past the budget gets one short line, which is not catalog text.
	require.LessOrEqual(t, total-notes, maxReviewToolBytes, "the catalog text stays within the budget")
}

func TestCatalogSizesTheBudgetToThePrompt(t *testing.T) {
	t.Parallel()

	budget := func(option CatalogOption) int {
		tools, ok := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{}, option).(*catalogTools)
		require.True(t, ok)
		return tools.budget
	}
	require.Equal(t, maxReviewToolBytes, budget(withPromptBytes(1<<10)), "a small prompt leaves the most")
	require.Equal(t, reviewTextBytes-(300<<10), budget(withPromptBytes(300<<10)))
	require.Equal(t, -1, budget(withPromptBytes(400<<10)), "a prompt that takes the whole share leaves no catalog text")
	require.Equal(t, -1, budget(withPromptBytes(reviewTextBytes)), "a budget of exactly zero is none, not the default")

	// The statement counts as the prompt writes it, every line with its number.
	shortLines := strings.Repeat("x\n", 100<<10)
	numbered, _ := numberLines(shortLines)
	require.Greater(t, len(numbered), 3*len(shortLines))
	require.Equal(t, budget(withPromptBytes(len(numbered)+len("policy"))), budget(WithPromptOf(&Request{Statement: shortLines, WorkspacePolicy: "policy"})))
}

func TestCatalogFitsAnEscapedDefinitionInItsLimit(t *testing.T) {
	t.Parallel()

	// Every quote and backslash doubles when encoded.
	escaped := strings.Repeat(`"\`, maxToolResultBytes)
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "PUBLIC", Functions: []*metadatapb.FunctionMetadata{{Name: "f", Definition: escaped}},
	}}})
	output, err := tools.Call(context.Background(), toolRead, `{"objects": [{"name": "f"}]}`)
	require.NoError(t, err)
	require.LessOrEqual(t, len(output), maxToolResultBytes+resultEnvelopeBytes)
	require.Contains(t, output, "next_from")

	spent := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{}, withPromptBytes(reviewTextBytes))
	first, err := spent.Call(context.Background(), toolRead, `{"objects": [{"name": "f"}]}`)
	require.NoError(t, err)
	require.Contains(t, first, "list in notes")
	later, err := spent.Call(context.Background(), toolSearch, `{}`)
	require.NoError(t, err)
	require.Less(t, len(later), len(`{"name": "read", "arguments": "{}"}`), "a later note is shorter than any call that asks for it")
}

func TestCatalogReadPagesThroughInvalidUTF8(t *testing.T) {
	t.Parallel()

	definition := "a" + strings.Repeat("\x80", 2*maxToolResultBytes) + "z"
	tools := unbounded(NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "PUBLIC", Functions: []*metadatapb.FunctionMetadata{{Name: "f", Definition: definition}},
	}}}))

	covered, from := 0, 0
	for pages := 0; ; pages++ {
		require.Less(t, pages, 50, "the pages must end")
		result := read(t, tools, fmt.Sprintf(`{"objects": [{"name": "f"}], "from": %d}`, from))
		require.Len(t, result.Objects, 1)
		var begin, stop, size int
		_, err := fmt.Sscanf(result.Objects[0].Part, "bytes %d to %d of %d", &begin, &stop, &size)
		require.NoError(t, err, result.Objects[0].Part)
		require.Equal(t, covered, begin, "each page starts where the last one stopped")
		covered = stop
		if result.NextFrom == nil {
			break
		}
		from = *result.NextFrom
	}
	require.Equal(t, len(definition), covered, "every byte is read")
}

func TestRuneStartKeepsEveryByteThatIsNoCharacter(t *testing.T) {
	t.Parallel()

	invalid := strings.Repeat("\x80", 100)
	require.Equal(t, 98, runeStart(invalid, 98), "a stray byte near the end is not skipped")
	require.Equal(t, 100, runeStart(invalid, 100))

	text := "a世b" // 世 is three bytes, at offsets 1 to 3.
	require.Equal(t, 1, runeStart(text, 1))
	require.Equal(t, 4, runeStart(text, 2), "inside a character, skip to its end")
	require.Equal(t, 4, runeStart(text, 3))
	require.Equal(t, 4, runeStart(text, 4))
}
