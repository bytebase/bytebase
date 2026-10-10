package aireview

import (
	"context"
	jsonv2 "encoding/json/v2"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	// The tools render definitions with the engines' renderers.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/pg"
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
	require.Equal(t, &tableStatistics{Rows: 52000000, DataBytes: 9 << 30, IndexBytes: 1 << 30}, orders.Statistics)
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
	require.Equal(t, &tableStatistics{Rows: 1200}, customers.Statistics)
	archived := read(t, pgShop(), `{"objects": [{"schema": "archive", "name": "orders"}]}`).Objects[0]
	require.Nil(t, archived.Statistics, "a table with no statistics synced shows none, zeros would say it is empty")
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
	require.Equal(t, &tableStatistics{Rows: 52000000, DataBytes: 9 << 30, IndexBytes: 1 << 30}, orders.Statistics)
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
		"PRIMARY KEY orders_pkey (id)",
		"INDEX idx_orders_customer (customer_id)",
		"",
		"FOREIGN KEY orders_customer_fk (customer_id) REFERENCES PUBLIC.customers (id)",
		"",
		"COMMENT: one row per order",
		"",
		"COMMENT ON COLUMN legacy_status: replaced by status",
	}, "\n"), result.Objects[0].Definition)
	require.Equal(t, "VIEW PUBLIC.revenue_by_status AS\nSELECT legacy_status, count(*) AS order_count FROM orders GROUP BY legacy_status", result.Objects[1].Definition)
	require.Equal(t, "SEQUENCE PUBLIC.orders_id_seq AS bigint START 1 INCREMENT 1 OWNED BY orders.id", result.Objects[2].Definition)
	require.Equal(t, "ENUM TYPE PUBLIC.payment_state: pending, paid", result.Objects[3].Definition)
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
	tools := NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{schema}})

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
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:      "PUBLIC",
		Functions: []*metadatapb.FunctionMetadata{{Name: "settle", Definition: definition}},
		Views:     []*metadatapb.ViewMetadata{{Name: "settled", Definition: "SELECT 1"}},
	}}})

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
		"ON PARTITION payments_2026:\n  INDEX payments_2026_order_idx (order_id)",
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
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name:      "PUBLIC",
		Functions: []*metadatapb.FunctionMetadata{{Name: "minified", Definition: definition}},
	}}})

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
	tools := NewCatalogTools(storepb.Engine_SNOWFLAKE, database)

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
		storepb.Engine_SNOWFLAKE: "SELECT 1 AS day\n\nINDEX day (day)",
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
	require.Equal(t, "EVENT TRIGGER log_ddl ON ddl_command_end WHEN TAG IN (CREATE TABLE, ALTER TABLE) EXECUTE FUNCTION public.log_ddl()\nDISABLED", result.Objects[1].Definition)
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
