// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { ExprType, type SimpleExpr, type StringFactor } from "@/modules/cel";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseMetadataSchema,
  DatabaseSchema$,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import { createMaskingRuleTargetChecker } from "./maskingRuleTarget";

const mocks = vi.hoisted(() => ({ getDatabaseMetadata: vi.fn() }));
vi.mock("@/api", () => ({
  databaseServiceClientConnect: {
    getDatabaseMetadata: mocks.getDatabaseMetadata,
  },
}));

const databaseName = "instances/primary/databases/example";
const now = Date.parse("2026-11-17T12:00:00Z");
const tableName = "resource.table_name";
const columnName = "resource.column_name";
const schemaName = "resource.schema_name";
const equals = (factor: StringFactor, value: string): SimpleExpr => ({
  type: ExprType.Condition,
  operator: "_==_",
  args: [factor, value],
});
const and = (...args: SimpleExpr[]): SimpleExpr => ({
  type: ExprType.ConditionGroup,
  operator: "_&&_",
  args,
});
const tableAndColumn = and(
  equals(tableName, "staff"),
  equals(columnName, "email")
);
const database = () =>
  create(DatabaseSchema$, {
    name: databaseName,
    project: "projects/example",
    state: State.ACTIVE,
    syncStatus: SyncStatus.OK,
    successfulSyncTime: {
      seconds: BigInt((now - 60_000) / 1000),
      nanos: 0,
    },
  });
const metadata = (
  tables: { name: string; columns?: { name: string }[] }[] = [
    { name: "staff", columns: [{ name: "email" }] },
  ]
) =>
  create(DatabaseMetadataSchema, {
    name: `${databaseName}/metadata`,
    schemas: [{ name: "public", tables }],
  });

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
  vi.setSystemTime(now);
  mocks.getDatabaseMetadata.mockResolvedValue(metadata());
});

afterEach(() => vi.useRealTimers());

describe("createMaskingRuleTargetChecker", () => {
  it("observes the exact table and column without returning unrelated metadata", async () => {
    const value = metadata([
      { name: "staff", columns: [{ name: "email" }, { name: "private" }] },
    ]);
    value.schemas[0].comment = "sensitive metadata comment";
    mocks.getDatabaseMetadata.mockResolvedValue(value);

    const result = await createMaskingRuleTargetChecker()(
      database(),
      tableAndColumn
    );

    expect(result).toMatchObject({
      status: "found",
      source: "synchronized-table-metadata",
      checkedAt: "2026-11-17T12:00:00.000Z",
      database: databaseName,
      successfulSyncTime: "2026-11-17T11:59:00.000Z",
      target: { table: "staff", column: "email" },
    });
    expect(JSON.stringify(result)).not.toContain("sensitive metadata comment");
    expect(JSON.stringify(result)).not.toContain("private");
    const [request, options] = mocks.getDatabaseMetadata.mock.calls[0];
    expect(request).toMatchObject({
      name: `${databaseName}/metadata`,
      filter: 'table.contains("staff")',
      limit: 20,
    });
    expect(options.timeoutMs).toBe(5000);
    expect(options.contextValues.get(silentContextKey)).toBe(true);
  });

  it.each([
    ["table only", equals(tableName, "staff"), "found"],
    ["column only", equals(columnName, "email"), "found"],
    ["missing table", equals(tableName, "missing"), "no-current-match"],
    [
      "missing column",
      and(equals(tableName, "staff"), equals(columnName, "missing")),
      "no-current-match",
    ],
    ["wrong table case", equals(tableName, "Staff"), "no-current-match"],
    ["wrong column case", equals(columnName, "Email"), "no-current-match"],
  ] as const)(
    "classifies %s from exact names",
    async (_, condition, status) => {
      const result = await createMaskingRuleTargetChecker()(
        database(),
        condition
      );
      expect(result.status).toBe(status);
    }
  );

  it("finds uppercase names with a contains filter and exact local comparison", async () => {
    mocks.getDatabaseMetadata.mockResolvedValue(
      metadata([{ name: "Staff", columns: [{ name: "Email" }] }])
    );
    const result = await createMaskingRuleTargetChecker()(
      database(),
      and(equals(tableName, "Staff"), equals(columnName, "Email"))
    );
    expect(result.status).toBe("found");
    expect(mocks.getDatabaseMetadata.mock.calls[0][0].filter).toBe(
      'table.contains("Staff")'
    );
  });

  it("escapes table literals without changing the exact requested target", async () => {
    const name = 'staff"\\archive';
    mocks.getDatabaseMetadata.mockResolvedValue(metadata([{ name }]));
    const result = await createMaskingRuleTargetChecker()(
      database(),
      equals(tableName, name)
    );
    expect(result).toMatchObject({ status: "found", target: { table: name } });
    expect(mocks.getDatabaseMetadata.mock.calls[0][0].filter).toBe(
      'table.contains("staff\\"\\\\archive")'
    );
  });

  it("uses a bounded all-table scan for an exact column without inventing a table", async () => {
    const result = await createMaskingRuleTargetChecker()(
      database(),
      equals(columnName, "email")
    );
    expect(result).toMatchObject({
      status: "found",
      target: { column: "email" },
    });
    expect(mocks.getDatabaseMetadata.mock.calls[0][0]).toMatchObject({
      filter: 'table.contains("")',
      limit: 20,
    });
  });

  it("does not observe a column in another table as the exact requested target", async () => {
    mocks.getDatabaseMetadata.mockResolvedValue(
      metadata([
        { name: "staff", columns: [{ name: "id" }] },
        { name: "staff_archive", columns: [{ name: "email" }] },
      ])
    );
    expect(
      (await createMaskingRuleTargetChecker()(database(), tableAndColumn))
        .status
    ).toBe("no-current-match");
  });

  describe("schema targets", () => {
    it("does not observe a matching table and column in another schema", async () => {
      mocks.getDatabaseMetadata.mockResolvedValue(
        create(DatabaseMetadataSchema, {
          name: `${databaseName}/metadata`,
          schemas: [
            {
              name: "archive",
              tables: [{ name: "staff", columns: [{ name: "email" }] }],
            },
            {
              name: "public",
              tables: [{ name: "staff", columns: [{ name: "id" }] }],
            },
          ],
        })
      );

      const result = await createMaskingRuleTargetChecker()(
        database(),
        and(equals(schemaName, "public"), tableAndColumn)
      );

      expect(result).toMatchObject({
        status: "no-current-match",
        target: { schema: "public", table: "staff", column: "email" },
      });
      expect(JSON.stringify(result)).not.toContain("archive");
    });

    it.each(["Public", ""])(
      "preserves and observes exact schema spelling %j without a schema filter",
      async (schema) => {
        const value = metadata();
        value.schemas[0].name = schema;
        mocks.getDatabaseMetadata.mockResolvedValue(value);

        const result = await createMaskingRuleTargetChecker()(
          database(),
          and(equals(schemaName, schema), tableAndColumn)
        );

        expect(result).toMatchObject({
          status: "found",
          target: { schema, table: "staff", column: "email" },
        });
        expect(mocks.getDatabaseMetadata.mock.calls[0][0]).toMatchObject({
          filter: 'table.contains("staff")',
          limit: 20,
        });
      }
    );

    it.each(["Public", "missing", ""])(
      "does not substitute public for schema %j",
      async (schema) => {
        expect(
          (
            await createMaskingRuleTargetChecker()(
              database(),
              and(equals(schemaName, schema), tableAndColumn)
            )
          ).status
        ).toBe("no-current-match");
      }
    );

    it.each(["public", ""])(
      "observes an existing schema %j without requiring any table",
      async (schema) => {
        const value = metadata([]);
        value.schemas[0].name = schema;
        mocks.getDatabaseMetadata.mockResolvedValue(value);

        const result = await createMaskingRuleTargetChecker()(
          database(),
          equals(schemaName, schema)
        );

        expect(result).toMatchObject({
          status: "found",
          target: { schema },
        });
        expect(mocks.getDatabaseMetadata.mock.calls[0][0]).toMatchObject({
          filter: 'table.contains("")',
          limit: 20,
        });
      }
    );

    it("reports no current match for an absent schema-only target", async () => {
      expect(
        (
          await createMaskingRuleTargetChecker()(
            database(),
            equals(schemaName, "missing")
          )
        ).status
      ).toBe("no-current-match");
    });

    it("checks an exact column only within the requested schema without requiring a table", async () => {
      expect(
        (
          await createMaskingRuleTargetChecker()(
            database(),
            and(equals(schemaName, "archive"), equals(columnName, "email"))
          )
        ).status
      ).toBe("no-current-match");
      expect(mocks.getDatabaseMetadata.mock.calls[0][0].filter).toBe(
        'table.contains("")'
      );
    });

    it("does not apply a table cap from another schema to an exact schema target", async () => {
      const value = metadata(
        Array.from({ length: 20 }, (_, i) => ({ name: `staff_${i}` }))
      );
      value.schemas.push(
        create(DatabaseMetadataSchema, {
          schemas: [{ name: "empty", tables: [] }],
        }).schemas[0]
      );
      mocks.getDatabaseMetadata.mockResolvedValue(value);

      expect(
        (
          await createMaskingRuleTargetChecker()(
            database(),
            and(equals(schemaName, "empty"), tableAndColumn)
          )
        ).status
      ).toBe("no-current-match");
    });

    it("does not claim schema absence beyond the local schema limit", async () => {
      const value = metadata([]);
      value.schemas = Array.from(
        { length: 101 },
        (_, i) =>
          create(DatabaseMetadataSchema, {
            schemas: [{ name: `schema_${i}`, tables: [] }],
          }).schemas[0]
      );
      mocks.getDatabaseMetadata.mockResolvedValue(value);

      const result = await createMaskingRuleTargetChecker()(
        database(),
        equals(schemaName, "schema_100")
      );

      expect(result.status).toBe("unverified");
      expect(mocks.getDatabaseMetadata).toHaveBeenCalledTimes(1);
    });

    it.each([
      {
        type: ExprType.Condition,
        operator: "startsWith",
        args: [schemaName, "pub"],
      },
      {
        type: ExprType.Condition,
        operator: "@in",
        args: [schemaName, ["public"]],
      },
      {
        type: ExprType.Condition,
        operator: "_!=_",
        args: [schemaName, "archive"],
      },
      and(equals(schemaName, "public"), equals(schemaName, "archive")),
      {
        type: ExprType.ConditionGroup,
        operator: "_||_",
        args: [equals(schemaName, "public"), equals(schemaName, "archive")],
      },
    ] as const)(
      "does not ignore an unsupported schema restriction: %j",
      async (restriction) => {
        const result = await createMaskingRuleTargetChecker()(
          database(),
          and(restriction as SimpleExpr, tableAndColumn)
        );

        expect(result.status).toBe("unverified");
        expect(mocks.getDatabaseMetadata).not.toHaveBeenCalled();
      }
    );

    it("shares a metadata read without sharing observations across schemas", async () => {
      const value = metadata();
      value.schemas.push(
        create(DatabaseMetadataSchema, {
          schemas: [
            {
              name: "archive",
              tables: [{ name: "staff", columns: [{ name: "id" }] }],
            },
          ],
        }).schemas[0]
      );
      mocks.getDatabaseMetadata.mockResolvedValue(value);
      const checker = createMaskingRuleTargetChecker();

      const found = await checker(
        database(),
        and(equals(schemaName, "public"), tableAndColumn)
      );
      const missing = await checker(
        database(),
        and(equals(schemaName, "archive"), tableAndColumn)
      );

      expect(found.status).toBe("found");
      expect(missing.status).toBe("no-current-match");
      expect(mocks.getDatabaseMetadata).toHaveBeenCalledTimes(1);
    });
  });

  it("does not claim a full rule match from a named target observation", async () => {
    const result = await createMaskingRuleTargetChecker()(
      database(),
      and(tableAndColumn, {
        type: ExprType.Condition,
        operator: "_>=_",
        args: ["resource.classification_level", 3],
      })
    );
    expect(result.status).toBe("found");
    expect(result.reason).toMatch(/synchronized.*metadata/i);
    expect(result.reason).not.toMatch(/effective masking/i);
  });

  it.each([
    ["unscoped database", undefined, tableAndColumn],
    [
      "classification only",
      database(),
      {
        type: ExprType.Condition,
        operator: "_>=_",
        args: ["resource.classification_level", 3],
      },
    ],
    [
      "pattern",
      database(),
      {
        type: ExprType.Condition,
        operator: "contains",
        args: [columnName, "mail"],
      },
    ],
    [
      "table with column pattern",
      database(),
      and(equals(tableName, "staff"), {
        type: ExprType.Condition,
        operator: "contains",
        args: [columnName, "mail"],
      }),
    ],
    [
      "negative",
      database(),
      {
        type: ExprType.Condition,
        operator: "_!=_",
        args: [columnName, "email"],
      },
    ],
    [
      "disjunction",
      database(),
      {
        type: ExprType.ConditionGroup,
        operator: "_||_",
        args: [tableAndColumn, equals(columnName, "name")],
      },
    ],
    [
      "nested disjunction",
      database(),
      and(equals(columnName, "email"), {
        type: ExprType.ConditionGroup,
        operator: "_||_",
        args: [equals(tableName, "staff"), equals(tableName, "staff_archive")],
      }),
    ],
    [
      "conflicting exact targets",
      database(),
      and(equals(tableName, "staff"), equals(tableName, "other")),
    ],
    [
      "raw expression",
      database(),
      { type: ExprType.RawString, content: 'resource.column_name == "email"' },
    ],
    [
      "finite collection",
      database(),
      {
        type: ExprType.Condition,
        operator: "@in",
        args: [columnName, ["email", "name"]],
      },
    ],
  ] as const)("does not scan %s", async (_, db, expression) => {
    const result = await createMaskingRuleTargetChecker()(
      db,
      expression as SimpleExpr
    );
    expect(result.status).toBe("unverified");
    expect(mocks.getDatabaseMetadata).not.toHaveBeenCalled();
  });

  it.each([
    [
      "failed sync",
      (db: ReturnType<typeof database>) => {
        db.syncStatus = SyncStatus.FAILED;
      },
    ],
    [
      "unknown sync",
      (db: ReturnType<typeof database>) => {
        db.syncStatus = SyncStatus.SYNC_STATUS_UNSPECIFIED;
      },
    ],
    [
      "missing timestamp",
      (db: ReturnType<typeof database>) => {
        db.successfulSyncTime = undefined;
      },
    ],
    [
      "stale timestamp",
      (db: ReturnType<typeof database>) => {
        db.successfulSyncTime!.seconds = BigInt(
          Math.floor((now - 86_400_001) / 1000)
        );
      },
    ],
    [
      "future timestamp",
      (db: ReturnType<typeof database>) => {
        db.successfulSyncTime!.seconds = BigInt(now / 1000 + 1);
      },
    ],
    [
      "invalid nanos",
      (db: ReturnType<typeof database>) => {
        db.successfulSyncTime!.nanos = 1_000_000_000;
      },
    ],
    [
      "invalid timestamp",
      (db: ReturnType<typeof database>) => {
        db.successfulSyncTime!.seconds = 999999999999999999999999n;
      },
    ],
  ])("does not read metadata with %s", async (_, mutate) => {
    const db = database();
    mutate(db);
    const result = await createMaskingRuleTargetChecker()(db, tableAndColumn);
    expect(result.status).toBe("unverified");
    expect(mocks.getDatabaseMetadata).not.toHaveBeenCalled();
  });

  it("accepts sync evidence exactly one day old", async () => {
    const db = database();
    db.successfulSyncTime!.seconds = BigInt((now - 86_400_000) / 1000);
    expect(
      (await createMaskingRuleTargetChecker()(db, tableAndColumn)).status
    ).toBe("found");
  });

  it.each([
    Code.PermissionDenied,
    Code.NotFound,
    Code.DeadlineExceeded,
    Code.Internal,
  ])("returns compact unverified evidence for API code %s", async (code) => {
    mocks.getDatabaseMetadata.mockRejectedValue(
      new ConnectError("sensitive error payload", code)
    );
    const result = await createMaskingRuleTargetChecker()(
      database(),
      tableAndColumn
    );
    expect(result.status).toBe("unverified");
    expect(JSON.stringify(result)).not.toContain("sensitive error payload");
  });

  it("rejects metadata identity from another database", async () => {
    const value = metadata();
    value.name = "instances/other/databases/example/metadata";
    mocks.getDatabaseMetadata.mockResolvedValue(value);
    expect(
      (await createMaskingRuleTargetChecker()(database(), tableAndColumn))
        .status
    ).toBe("unverified");
  });

  it("treats a per-schema table cap as incomplete when no target was observed", async () => {
    mocks.getDatabaseMetadata.mockResolvedValue(
      metadata(Array.from({ length: 20 }, (_, i) => ({ name: `staff_${i}` })))
    );
    expect(
      (await createMaskingRuleTargetChecker()(database(), tableAndColumn))
        .status
    ).toBe("unverified");
  });

  it("accepts an observed exact match even when a table cap prevents an absence claim", async () => {
    mocks.getDatabaseMetadata.mockResolvedValue(
      metadata([
        { name: "staff", columns: [{ name: "email" }] },
        ...Array.from({ length: 19 }, (_, i) => ({ name: `staff_${i}` })),
      ])
    );
    expect(
      (await createMaskingRuleTargetChecker()(database(), tableAndColumn))
        .status
    ).toBe("found");
  });

  it("stops local schema processing without claiming absence", async () => {
    const value = metadata([]);
    value.schemas = Array.from(
      { length: 101 },
      (_, i) =>
        create(DatabaseMetadataSchema, {
          schemas: [{ name: `schema_${i}`, tables: [] }],
        }).schemas[0]
    );
    value.schemas[100].tables = metadata().schemas[0].tables;
    mocks.getDatabaseMetadata.mockResolvedValue(value);
    expect(
      (await createMaskingRuleTargetChecker()(database(), tableAndColumn))
        .status
    ).toBe("unverified");
  });

  it("stops local column processing without claiming absence", async () => {
    mocks.getDatabaseMetadata.mockResolvedValue(
      metadata([
        {
          name: "staff",
          columns: [
            ...Array.from({ length: 20_000 }, (_, i) => ({
              name: `column_${i}`,
            })),
            { name: "email" },
          ],
        },
      ])
    );
    expect(
      (await createMaskingRuleTargetChecker()(database(), tableAndColumn))
        .status
    ).toBe("unverified");
  });

  it("ignores external tables and views as evidence for a synchronized table target", async () => {
    const value = metadata([]);
    const unsupported = create(DatabaseMetadataSchema, {
      schemas: [
        {
          externalTables: [{ name: "staff" }],
          views: [{ name: "staff", columns: [{ name: "email" }] }],
        },
      ],
    }).schemas[0];
    value.schemas[0].views = unsupported.views;
    value.schemas[0].externalTables = unsupported.externalTables;
    mocks.getDatabaseMetadata.mockResolvedValue(value);
    const result = await createMaskingRuleTargetChecker()(
      database(),
      tableAndColumn
    );
    expect(result.status).toBe("no-current-match");
    expect(result.reason).toMatch(/table metadata/i);
    expect(result.reason).not.toMatch(/does not exist/i);
  });

  it("shares metadata promises only within one preparation checker", async () => {
    const checker = createMaskingRuleTargetChecker();
    const [found, missing] = await Promise.all([
      checker(database(), tableAndColumn),
      checker(
        database(),
        and(equals(tableName, "staff"), equals(columnName, "missing"))
      ),
    ]);
    expect(found.status).toBe("found");
    expect(missing.status).toBe("no-current-match");
    expect(mocks.getDatabaseMetadata).toHaveBeenCalledTimes(1);

    await createMaskingRuleTargetChecker()(database(), tableAndColumn);
    expect(mocks.getDatabaseMetadata).toHaveBeenCalledTimes(2);
  });

  it("does not share metadata between different database identities", async () => {
    const checker = createMaskingRuleTargetChecker();
    const other = database();
    other.name = "instances/other/databases/example";
    mocks.getDatabaseMetadata
      .mockResolvedValueOnce(metadata())
      .mockResolvedValueOnce(
        create(DatabaseMetadataSchema, { name: `${other.name}/metadata` })
      );
    const [found, missing] = await Promise.all([
      checker(database(), tableAndColumn),
      checker(other, tableAndColumn),
    ]);
    expect(found.status).toBe("found");
    expect(missing.status).toBe("no-current-match");
    expect(mocks.getDatabaseMetadata).toHaveBeenCalledTimes(2);
  });
});
