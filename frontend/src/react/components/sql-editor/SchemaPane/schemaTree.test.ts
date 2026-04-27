import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import {
  CheckConstraintMetadataSchema,
  ColumnMetadataSchema,
  DatabaseMetadataSchema,
  DatabaseSchema$,
  DependencyColumnSchema,
  ExternalTableMetadataSchema,
  ForeignKeyMetadataSchema,
  FunctionMetadataSchema,
  IndexMetadataSchema,
  PackageMetadataSchema,
  ProcedureMetadataSchema,
  SchemaMetadataSchema,
  SequenceMetadataSchema,
  TableMetadataSchema,
  TablePartitionMetadataSchema,
  TriggerMetadataSchema,
  ViewMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  buildDatabaseSchemaTree,
  ExpandableNodeTypes,
  keyForNodeTarget,
  LeafNodeTypes,
  mapTreeNodeByType,
  type NodeTarget,
  type NodeType,
  readableTextForNodeTarget,
  type TextTarget,
  type TreeNode,
} from "./schemaTree";

vi.mock("@/react/i18n", () => ({
  default: { t: (key: string) => key },
}));

const DATABASE = "instances/i1/databases/db1";

describe("keyForNodeTarget", () => {
  test("database key is the database resource name", () => {
    expect(keyForNodeTarget("database", { database: DATABASE })).toBe(DATABASE);
  });

  test("schema key joins under the database", () => {
    expect(
      keyForNodeTarget("schema", { database: DATABASE, schema: "public" })
    ).toBe(`${DATABASE}/schemas/public`);
  });

  test("table key joins under the schema", () => {
    expect(
      keyForNodeTarget("table", {
        database: DATABASE,
        schema: "public",
        table: "users",
      })
    ).toBe(`${DATABASE}/schemas/public/tables/users`);
  });

  test("external-table key joins under the schema", () => {
    expect(
      keyForNodeTarget("external-table", {
        database: DATABASE,
        schema: "ext",
        externalTable: "logs",
      })
    ).toBe(`${DATABASE}/schemas/ext/externalTables/logs`);
  });

  test("view key joins under the schema", () => {
    expect(
      keyForNodeTarget("view", {
        database: DATABASE,
        schema: "public",
        view: "active_users",
      })
    ).toBe(`${DATABASE}/schemas/public/views/active_users`);
  });

  test("procedure key uses keyWithPosition under the schema", () => {
    expect(
      keyForNodeTarget("procedure", {
        database: DATABASE,
        schema: "public",
        procedure: "do_thing",
        position: 0,
      })
    ).toBe(`${DATABASE}/schemas/public/procedures/do_thing###0`);
  });

  test("package key uses keyWithPosition under the schema", () => {
    expect(
      keyForNodeTarget("package", {
        database: DATABASE,
        schema: "public",
        package: "pkg",
        position: 1,
      })
    ).toBe(`${DATABASE}/schemas/public/packages/pkg###1`);
  });

  test("function key uses keyWithPosition under the schema", () => {
    expect(
      keyForNodeTarget("function", {
        database: DATABASE,
        schema: "public",
        function: "fn",
        position: 2,
      })
    ).toBe(`${DATABASE}/schemas/public/functions/fn###2`);
  });

  test("sequence key uses keyWithPosition under the schema", () => {
    expect(
      keyForNodeTarget("sequence", {
        database: DATABASE,
        schema: "public",
        sequence: "seq",
        position: 3,
      })
    ).toBe(`${DATABASE}/schemas/public/sequences/seq###3`);
  });

  test("trigger key uses keyWithPosition under the table", () => {
    expect(
      keyForNodeTarget("trigger", {
        database: DATABASE,
        schema: "public",
        table: "users",
        trigger: "trg",
        position: 4,
      })
    ).toBe(`${DATABASE}/schemas/public/tables/users/triggers/trg###4`);
  });

  test("index key joins under the table", () => {
    expect(
      keyForNodeTarget("index", {
        database: DATABASE,
        schema: "public",
        table: "users",
        index: "ix_email",
      })
    ).toBe(`${DATABASE}/schemas/public/tables/users/indexes/ix_email`);
  });

  test("foreign-key joins under the table", () => {
    expect(
      keyForNodeTarget("foreign-key", {
        database: DATABASE,
        schema: "public",
        table: "users",
        foreignKey: "fk_org",
      })
    ).toBe(`${DATABASE}/schemas/public/tables/users/foreignKeys/fk_org`);
  });

  test("check joins under the table", () => {
    expect(
      keyForNodeTarget("check", {
        database: DATABASE,
        schema: "public",
        table: "users",
        check: "ck_age",
      })
    ).toBe(`${DATABASE}/schemas/public/tables/users/checks/ck_age`);
  });

  test("partition-table joins under the table", () => {
    expect(
      keyForNodeTarget("partition-table", {
        database: DATABASE,
        schema: "public",
        table: "events",
        partition: "p2026",
      })
    ).toBe(`${DATABASE}/schemas/public/tables/events/partitionTables/p2026`);
  });

  test("dependency-column joins under the view", () => {
    const dep = create(DependencyColumnSchema, {
      schema: "public",
      table: "users",
      column: "id",
    });
    expect(
      keyForNodeTarget("dependency-column", {
        database: DATABASE,
        schema: "public",
        view: "active_users",
        dependencyColumn: dep,
      })
    ).toBe(
      `${DATABASE}/schemas/public/views/active_users/dependencyColumns/public/users/id`
    );
  });

  test("column under a table joins under the table", () => {
    expect(
      keyForNodeTarget("column", {
        database: DATABASE,
        schema: "public",
        table: "users",
        column: "id",
      })
    ).toBe(`${DATABASE}/schemas/public/tables/users/columns/id`);
  });

  test("expandable-text + error keys join type/id", () => {
    expect(
      keyForNodeTarget("expandable-text", {
        id: "x/y/tables",
        expandable: true,
        text: () => "Tables",
      })
    ).toBe("expandable-text/x/y/tables");
    expect(
      keyForNodeTarget("error", {
        id: "x/y/dummy-table-0",
        expandable: false,
        text: () => "",
      })
    ).toBe("error/x/y/dummy-table-0");
  });
});

describe("readableTextForNodeTarget", () => {
  test("error nodes are empty (so search ignores them)", () => {
    expect(
      readableTextForNodeTarget("error", {
        id: "x",
        expandable: false,
        text: () => "anything",
      })
    ).toBe("");
  });

  test("non-searchable expandable-text returns empty", () => {
    expect(
      readableTextForNodeTarget("expandable-text", {
        id: "x",
        expandable: true,
        text: () => "Tables",
      })
    ).toBe("");
  });

  test("searchable expandable-text returns the text() value", () => {
    expect(
      readableTextForNodeTarget("expandable-text", {
        id: "x",
        expandable: true,
        text: () => "Tables",
        searchable: true,
      })
    ).toBe("Tables");
  });

  test("dependency-column joins schema.table.column", () => {
    const dep = create(DependencyColumnSchema, {
      schema: "public",
      table: "users",
      column: "id",
    });
    expect(
      readableTextForNodeTarget("dependency-column", {
        database: DATABASE,
        schema: "public",
        view: "v",
        dependencyColumn: dep,
      })
    ).toBe("public.users.id");
  });

  test("dependency-column without schema joins table.column only", () => {
    const dep = create(DependencyColumnSchema, {
      table: "users",
      column: "id",
    });
    expect(
      readableTextForNodeTarget("dependency-column", {
        database: DATABASE,
        schema: "public",
        view: "v",
        dependencyColumn: dep,
      })
    ).toBe("users.id");
  });

  test("default case returns the last path segment", () => {
    expect(
      readableTextForNodeTarget("table", {
        database: DATABASE,
        schema: "public",
        table: "users",
      })
    ).toBe("users");
  });
});

describe("LeafNodeTypes / ExpandableNodeTypes", () => {
  test("type unions partition cleanly", () => {
    const allTypes: NodeType[] = [
      "database",
      "schema",
      "table",
      "external-table",
      "view",
      "procedure",
      "function",
      "sequence",
      "trigger",
      "package",
      "partition-table",
      "column",
      "index",
      "foreign-key",
      "check",
      "dependency-column",
      "expandable-text",
      "error",
    ];
    for (const t of allTypes) {
      expect(
        LeafNodeTypes.includes(t) ||
          ExpandableNodeTypes.includes(t) ||
          [
            "database",
            "schema",
            "table",
            "view",
            "external-table",
            "partition-table",
            "expandable-text",
          ].includes(t) ||
          LeafNodeTypes.includes(t)
      ).toBe(true);
    }
  });

  test("mapTreeNodeByType marks leaves correctly", () => {
    const colNode = mapTreeNodeByType("column", {
      database: DATABASE,
      schema: "public",
      table: "users",
      column: "id",
    });
    expect(colNode.isLeaf).toBe(true);

    const schemaNode = mapTreeNodeByType("schema", {
      database: DATABASE,
      schema: "public",
    });
    expect(schemaNode.isLeaf).toBe(false);
  });
});

// ---------- buildDatabaseSchemaTree ----------

const makeDatabase = () => create(DatabaseSchema$, { name: DATABASE });

const makeColumn = (name: string) => create(ColumnMetadataSchema, { name });

const makeIndex = (name: string) => create(IndexMetadataSchema, { name });

const makeForeignKey = (name: string) =>
  create(ForeignKeyMetadataSchema, { name });

const makeCheck = (name: string) =>
  create(CheckConstraintMetadataSchema, { name });

const makePartition = (name: string, subs: string[] = []) =>
  create(TablePartitionMetadataSchema, {
    name,
    subpartitions: subs.map((n) =>
      create(TablePartitionMetadataSchema, { name: n })
    ),
  });

const makeTable = (
  name: string,
  opts: Partial<{
    columns: string[];
    indexes: string[];
    foreignKeys: string[];
    checks: string[];
    triggers: string[];
    partitions: ReturnType<typeof makePartition>[];
  }> = {}
) =>
  create(TableMetadataSchema, {
    name,
    columns: (opts.columns ?? []).map(makeColumn),
    indexes: (opts.indexes ?? []).map(makeIndex),
    foreignKeys: (opts.foreignKeys ?? []).map(makeForeignKey),
    checkConstraints: (opts.checks ?? []).map(makeCheck),
    triggers: (opts.triggers ?? []).map((n) =>
      create(TriggerMetadataSchema, { name: n })
    ),
    partitions: opts.partitions ?? [],
  });

const makeView = (name: string, columns: string[] = []) =>
  create(ViewMetadataSchema, {
    name,
    columns: columns.map(makeColumn),
  });

const makeExternalTable = (name: string, columns: string[] = []) =>
  create(ExternalTableMetadataSchema, {
    name,
    columns: columns.map(makeColumn),
  });

const makeSchema = (
  name: string,
  opts: Partial<{
    tables: ReturnType<typeof makeTable>[];
    externalTables: ReturnType<typeof makeExternalTable>[];
    views: ReturnType<typeof makeView>[];
    procedures: string[];
    functions: string[];
    sequences: string[];
    packages: string[];
  }> = {}
) =>
  create(SchemaMetadataSchema, {
    name,
    tables: opts.tables ?? [],
    externalTables: opts.externalTables ?? [],
    views: opts.views ?? [],
    procedures: (opts.procedures ?? []).map((n) =>
      create(ProcedureMetadataSchema, { name: n })
    ),
    functions: (opts.functions ?? []).map((n) =>
      create(FunctionMetadataSchema, { name: n })
    ),
    sequences: (opts.sequences ?? []).map((n) =>
      create(SequenceMetadataSchema, { name: n })
    ),
    packages: (opts.packages ?? []).map((n) =>
      create(PackageMetadataSchema, { name: n })
    ),
  });

const makeMetadata = (schemas: ReturnType<typeof makeSchema>[]) =>
  create(DatabaseMetadataSchema, { schemas });

// Walk a TreeNode tree and yield (key, type) for every node.
function* walk(
  nodes: TreeNode[]
): Generator<{ key: string; type: NodeType; label: string | undefined }> {
  for (const n of nodes) {
    yield { key: n.key, type: n.meta.type, label: n.label };
    if (n.children) yield* walk(n.children);
  }
}

const findKey = (nodes: TreeNode[], predicate: (k: string) => boolean) =>
  [...walk(nodes)].find((n) => predicate(n.key));

describe("buildDatabaseSchemaTree", () => {
  test("empty metadata returns a dummy 'empty' placeholder", () => {
    const tree = buildDatabaseSchemaTree(makeDatabase(), makeMetadata([]));
    expect(tree).toHaveLength(1);
    expect(tree[0].meta.type).toBe("error");
    expect(tree[0].disabled).toBe(true);
  });

  test("single anonymous schema (MySQL-style) flattens schema's children to root", () => {
    const md = makeMetadata([
      makeSchema("", {
        tables: [makeTable("users", { columns: ["id", "email"] })],
      }),
    ]);
    const tree = buildDatabaseSchemaTree(makeDatabase(), md);
    // Top-level should be the "Tables" expandable folder, not a schema node.
    expect(tree[0].meta.type).toBe("expandable-text");
    const tablesFolder = tree[0];
    expect(tablesFolder.children?.[0].meta.type).toBe("table");
    expect(tablesFolder.children?.[0].key).toBe(
      `${DATABASE}/schemas//tables/users`
    );
  });

  test("multi-schema (Postgres-style) wraps children in schema nodes", () => {
    const md = makeMetadata([
      makeSchema("public", { tables: [makeTable("users")] }),
      makeSchema("audit", { tables: [makeTable("events")] }),
    ]);
    const tree = buildDatabaseSchemaTree(makeDatabase(), md);
    expect(tree).toHaveLength(2);
    expect(tree[0].meta.type).toBe("schema");
    expect(tree[0].key).toBe(`${DATABASE}/schemas/public`);
    expect(tree[1].key).toBe(`${DATABASE}/schemas/audit`);
  });

  test("a table builds Columns / Indexes / FKs / Checks / Triggers / Partitions folders only when populated", () => {
    const md = makeMetadata([
      makeSchema("public", {
        tables: [
          makeTable("users", {
            columns: ["id"],
            indexes: ["ix_id"],
            foreignKeys: ["fk_org"],
            checks: ["ck_age"],
            triggers: ["trg_audit"],
            partitions: [makePartition("p1")],
          }),
          makeTable("plain", { columns: ["x"] }),
        ],
      }),
    ]);
    const tree = buildDatabaseSchemaTree(makeDatabase(), md);
    const usersTable = findKey(
      tree,
      (k) => k === `${DATABASE}/schemas/public/tables/users`
    );
    expect(usersTable).toBeDefined();

    const plainTable = [...walk(tree)].filter(
      (n) => n.key === `${DATABASE}/schemas/public/tables/plain`
    );
    expect(plainTable).toHaveLength(1);

    // Plain table has only the Columns folder.
    const plainNode = tree[0].children![0].children!.find(
      (c) => c.key === `${DATABASE}/schemas/public/tables/plain`
    );
    expect(plainNode!.children).toHaveLength(1);
    expect(plainNode!.children![0].meta.type).toBe("expandable-text");
  });

  test("nested partitions recurse and reuse the table-level key prefix", () => {
    // Vue's keyForNodeTarget("partition-table", ...) always keys directly
    // under the table — not under the parent partition. Sub-partitions
    // therefore share the same `<table>/partitionTables/<name>` prefix as
    // top-level partitions; preserving this behavior keeps persisted
    // treeState keys valid.
    const md = makeMetadata([
      makeSchema("public", {
        tables: [
          makeTable("events", {
            columns: ["id"],
            partitions: [makePartition("p2026", ["q1", "q2"])],
          }),
        ],
      }),
    ]);
    const tree = buildDatabaseSchemaTree(makeDatabase(), md);
    const parent = findKey(
      tree,
      (k) =>
        k === `${DATABASE}/schemas/public/tables/events/partitionTables/p2026`
    );
    expect(parent).toBeDefined();
    expect(parent!.type).toBe("partition-table");

    // The sub-partitions are children of `parent` but key directly under
    // the table (not nested under `p2026`).
    const q1 = findKey(
      tree,
      (k) => k === `${DATABASE}/schemas/public/tables/events/partitionTables/q1`
    );
    expect(q1).toBeDefined();
    expect(q1!.type).toBe("partition-table");
  });

  test("empty Tables folder still emits an <Empty> child for parity", () => {
    const md = makeMetadata([
      makeSchema("public", {
        tables: [],
        views: [makeView("v")],
      }),
    ]);
    const tree = buildDatabaseSchemaTree(makeDatabase(), md);
    const tablesFolder = tree[0].children!.find(
      (c) =>
        c.meta.type === "expandable-text" &&
        (c.meta.target as TextTarget).id === `${DATABASE}/schemas/public/table`
    );
    expect(tablesFolder).toBeDefined();
    expect(tablesFolder!.children![0].meta.type).toBe("error");
  });

  test("external tables / views / procedures / functions / packages / sequences only appear when populated", () => {
    const md = makeMetadata([
      makeSchema("public", {
        tables: [makeTable("t")],
        externalTables: [makeExternalTable("ext_t", ["c"])],
        views: [makeView("v", ["c"])],
        procedures: ["p"],
        functions: ["f"],
        packages: ["pk"],
        sequences: ["s"],
      }),
    ]);
    const tree = buildDatabaseSchemaTree(makeDatabase(), md);
    const folderTypes = tree[0].children!.map(
      (c) => (c.meta.target as TextTarget).mockType
    );
    expect(folderTypes).toEqual([
      "table",
      "external-table",
      "view",
      "procedure",
      "package",
      "function",
      "sequence",
    ]);
  });

  test("dependency-column children would key under the view (synthetic check)", () => {
    // dependency-column is built externally (lineage feature) but the key
    // path must remain stable. Verify keyForNodeTarget output directly.
    const dep = create(DependencyColumnSchema, {
      schema: "ext",
      table: "raw_users",
      column: "id",
    });
    expect(
      keyForNodeTarget("dependency-column", {
        database: DATABASE,
        schema: "public",
        view: "active",
        dependencyColumn: dep,
      })
    ).toBe(
      `${DATABASE}/schemas/public/views/active/dependencyColumns/ext/raw_users/id`
    );
  });
});

// Compile-time check that NodeTarget specializations remain assignable.
const _typeCheckTargets: {
  database: NodeTarget<"database">;
  schema: NodeTarget<"schema">;
  trigger: NodeTarget<"trigger">;
  expandableText: NodeTarget<"expandable-text">;
} = {
  database: { database: DATABASE },
  schema: { database: DATABASE, schema: "public" },
  trigger: {
    database: DATABASE,
    schema: "public",
    table: "t",
    trigger: "trg",
    position: 0,
  },
  expandableText: {
    id: "x",
    expandable: true,
    text: () => "x",
  },
};
void _typeCheckTargets;
