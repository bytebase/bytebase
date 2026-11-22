// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AppStoreState } from "@/stores/app/types";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseMetadataSchema,
  DatabaseSchema$,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import {
  consumeMaskingExemptionDraft,
  getMaskingExemptionDraft,
  getMaskingExemptionDraftSource,
  parseMaskingExemptionDraftCondition,
  prepareMaskingExemptionDraft,
  revalidateMaskingExemptionDraft,
  validateMaskingExemptionDraft,
} from "./maskingExemptionDraft";

const mocks = vi.hoisted(() => ({
  state: undefined as unknown as AppStoreState,
  setup: vi.fn(),
  project: vi.fn(),
  database: vi.fn(),
  metadata: vi.fn(),
  user: vi.fn(),
  policy: vi.fn(),
  parse: vi.fn(),
  deparse: vi.fn(),
  snapshot: vi.fn(),
  refresh: vi.fn(),
}));
vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => mocks.state },
}));
vi.mock("@/api", () => ({
  projectServiceClientConnect: { getProject: mocks.project },
  databaseServiceClientConnect: {
    getDatabase: mocks.database,
    getDatabaseMetadata: mocks.metadata,
  },
  userServiceClientConnect: { getUser: mocks.user },
  orgPolicyServiceClientConnect: { getPolicy: mocks.policy },
  celServiceClientConnect: {
    batchParse: mocks.parse,
    batchDeparse: mocks.deparse,
  },
}));
vi.mock("./tools/maskingSetup", () => ({ getMaskingSetup: mocks.setup }));
vi.mock("./maskingExemptionDraftTarget", () => ({
  getMaskingExemptionEditorSnapshot: mocks.snapshot,
}));
const context = {
  workspaceName: "workspaces/example",
  userName: "users/member@example.com",
  chatId: "chat",
};
const project = "projects/example";
const database = "instances/primary/databases/app";
const condition = 'resource.column_name == "email"';
const input = {
  project,
  scope: { kind: "database", database },
  members: ["self"],
  condition,
  description: "Temporary review",
  expirationTimestamp: "2030-01-01T00:00:00Z",
};
const ident = (name: string) =>
  create(ExprSchema, { exprKind: { case: "identExpr", value: { name } } });
const literal = (value: string) =>
  create(ExprSchema, {
    exprKind: {
      case: "constExpr",
      value: { constantKind: { case: "stringValue", value } },
    },
  });
const call = (fn: string, args: Expr[], target?: Expr) =>
  create(ExprSchema, {
    exprKind: { case: "callExpr", value: { function: fn, args, target } },
  });
const select = (field: string) =>
  create(ExprSchema, {
    exprKind: {
      case: "selectExpr",
      value: { operand: ident("resource"), field },
    },
  });
const expressions = new Map<string, Expr>();
beforeEach(() => {
  vi.clearAllMocks();
  mocks.state = {
    currentUser: {
      name: context.userName,
      email: "member@example.com",
      state: State.ACTIVE,
    },
    serverInfo: { workspace: context.workspaceName },
    refreshSubscription: mocks.refresh,
  } as unknown as AppStoreState;
  mocks.refresh.mockResolvedValue({});
  const evidence = {
    status: "allowed",
    requiredPermissions: [],
    missingPermissions: [],
    scope: project,
  };
  mocks.setup.mockResolvedValue({
    features: { masking: "available" },
    steps: [
      { id: "masking-exemptions", readAccess: evidence, configure: evidence },
    ],
  });
  mocks.project.mockResolvedValue({ name: project, state: State.ACTIVE });
  mocks.database.mockImplementation(async ({ name }) => ({
    name,
    project,
    state: State.ACTIVE,
  }));
  mocks.user.mockImplementation(async ({ name }) => ({
    name,
    email: name.slice(6),
    state: State.ACTIVE,
  }));
  mocks.policy.mockResolvedValue({
    policy: { case: "maskingExemptionPolicy", value: { exemptions: [] } },
  });
  mocks.snapshot.mockResolvedValue(undefined);
  expressions.clear();
  expressions.set(
    condition,
    call("_==_", [ident("resource.column_name"), literal("email")])
  );
  mocks.parse.mockImplementation(async ({ expressions: texts }) => ({
    expressions: texts.map((text: string) => expressions.get(text)),
  }));
  mocks.deparse.mockImplementation(async ({ expressions: exprs }) => ({
    expressions: exprs.map((expr: Expr) => {
      const text = `converted-${expressions.size}`;
      expressions.set(text, expr);
      return text;
    }),
  }));
});

afterEach(() => {
  vi.useRealTimers();
});

describe("exact exemption target evidence", () => {
  const canonical = `${project}/instances/primary/databases/app`;
  const targetCondition =
    'resource.schema_name == "public" && resource.table_name == "staff" && resource.column_name == "email"';
  const equals = (field: string, value: string) =>
    call("_==_", [select(field), literal(value)]);
  const target = () =>
    call("_&&_", [
      equals("schema_name", "public"),
      call("_&&_", [
        equals("table_name", "staff"),
        equals("column_name", "email"),
      ]),
    ]);
  const exact = () => ({
    ...input,
    condition: targetCondition,
    scope: { kind: "database", database: canonical },
  });
  const metadata = () =>
    create(DatabaseMetadataSchema, {
      name: `${canonical}/metadata`,
      schemas: [
        {
          name: "public",
          tables: [{ name: "staff", columns: [{ name: "email" }] }],
        },
      ],
    });

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-18T08:00:00Z"));
    expressions.set(targetCondition, target());
    mocks.database.mockImplementation(async ({ name }) =>
      create(DatabaseSchema$, {
        name,
        project,
        state: State.ACTIVE,
        syncStatus: SyncStatus.OK,
        successfulSyncTime: timestampFromDate(new Date()),
      })
    );
    mocks.metadata.mockResolvedValue(metadata());
  });

  it("automatically checks the exact schema target using the already verified canonical database", async () => {
    const result = await prepareMaskingExemptionDraft(exact(), context);

    expect(result).toMatchObject({
      draftId: expect.any(String),
      targetEvidence: {
        status: "found",
        database: canonical,
        target: { schema: "public", table: "staff", column: "email" },
      },
    });
    expect(mocks.database).toHaveBeenCalledTimes(1);
    expect(mocks.metadata).toHaveBeenCalledWith(
      expect.objectContaining({
        name: `${canonical}/metadata`,
        filter: 'table.contains("staff")',
        limit: 20,
      }),
      expect.anything()
    );
    expect(result.draft).not.toHaveProperty("targetEvidence");
  });

  it("requires clarification when only another schema has the same table and column", async () => {
    const value = metadata();
    value.schemas[0].name = "private";
    mocks.metadata.mockResolvedValue(value);

    await expect(
      prepareMaskingExemptionDraft(exact(), context)
    ).rejects.toMatchObject({
      targetEvidence: [
        expect.objectContaining({
          status: "no-current-match",
          target: { schema: "public", table: "staff", column: "email" },
        }),
      ],
    });
  });

  it("allows an acknowledged future target without storing acknowledgment or evidence in the exemption", async () => {
    mocks.metadata.mockResolvedValue(
      create(DatabaseMetadataSchema, { name: `${canonical}/metadata` })
    );

    const result = await prepareMaskingExemptionDraft(
      { ...exact(), allowUnmatchedTarget: true },
      context
    );

    expect(result).toMatchObject({
      targetEvidence: { status: "no-current-match" },
      draft: {
        scope: { kind: "database", database: canonical },
        members: ["user:member@example.com"],
        expirationTimestamp: "2030-01-01T00:00:00.000Z",
      },
    });
    const stored = getMaskingExemptionDraft(result.draftId, context);
    expect(stored).toEqual(result.draft);
    expect(stored).not.toHaveProperty("allowUnmatchedTarget");
    expect(stored).not.toHaveProperty("targetEvidence");
    const group = await parseMaskingExemptionDraftCondition(
      result.draft.condition
    );
    expect(group.args).toContainEqual({
      type: "Condition",
      operator: "_==_",
      args: ["resource.schema_name", "public"],
    });
  });

  it("returns unverified evidence instead of treating denied metadata as absence", async () => {
    mocks.metadata.mockRejectedValue(
      new ConnectError("denied", Code.PermissionDenied)
    );

    expect(await prepareMaskingExemptionDraft(exact(), context)).toMatchObject({
      draftId: expect.any(String),
      targetEvidence: { status: "unverified" },
    });
  });

  it("checks a project-scoped predicate constrained to one exact verified database", async () => {
    expressions.set(
      "single-database",
      call("_&&_", [equals("database", canonical), target()])
    );

    const result = await prepareMaskingExemptionDraft(
      { ...exact(), scope: { kind: "project" }, condition: "single-database" },
      context
    );

    expect(result).toMatchObject({
      targetEvidence: { status: "found", database: canonical },
      draft: { scope: { kind: "project" } },
    });
    expect(mocks.database).toHaveBeenCalledTimes(1);
    expect(mocks.metadata).toHaveBeenCalledTimes(1);
  });

  it.each([targetCondition, "true"])(
    "does not enumerate project-wide target metadata for %s",
    async (condition) => {
      const result = await prepareMaskingExemptionDraft(
        { ...exact(), scope: { kind: "project" }, condition },
        context
      );

      expect(result).toMatchObject({
        draftId: expect.any(String),
        targetEvidence: { status: "unverified" },
      });
      expect(mocks.metadata).not.toHaveBeenCalled();
      expect(mocks.database).not.toHaveBeenCalled();
    }
  );

  it("does not repeat target reads during ordinary draft validation or revalidation", async () => {
    const result = await prepareMaskingExemptionDraft(exact(), context);
    expect(mocks.metadata).toHaveBeenCalledTimes(1);
    mocks.metadata.mockClear();

    await validateMaskingExemptionDraft(result.draft, context);
    await revalidateMaskingExemptionDraft(result.draftId, context);

    expect(mocks.metadata).not.toHaveBeenCalled();
  });

  it.each(["disjunction", "multiple-databases"])(
    "does not treat %s as one exact database target",
    async (kind) => {
      expressions.set(
        kind,
        kind === "disjunction"
          ? call("_||_", [equals("database", canonical), target()])
          : call("_&&_", [
              call("@in", [
                select("database"),
                create(ExprSchema, {
                  exprKind: {
                    case: "listExpr",
                    value: {
                      elements: [
                        literal(canonical),
                        literal(`${project}/instances/primary/databases/other`),
                      ],
                    },
                  },
                }),
              ]),
              target(),
            ])
      );

      const result = await prepareMaskingExemptionDraft(
        { ...exact(), scope: { kind: "project" }, condition: kind },
        context
      );

      expect(result).toMatchObject({
        draftId: expect.any(String),
        targetEvidence: { status: "unverified" },
      });
      expect(result.targetEvidence).not.toHaveProperty("database");
      expect(mocks.metadata).not.toHaveBeenCalled();
    }
  );

  it("does not scan metadata for a name-pattern target within an exact database", async () => {
    expressions.set(
      "prefix",
      call("_&&_", [
        equals("schema_name", "public"),
        call("startsWith", [literal("private_")], select("column_name")),
      ])
    );

    const result = await prepareMaskingExemptionDraft(
      { ...exact(), condition: "prefix" },
      context
    );

    expect(result).toMatchObject({
      targetEvidence: { status: "unverified", database: canonical },
    });
    expect(mocks.metadata).not.toHaveBeenCalled();
  });

  it("rejects a malformed future-target acknowledgment", async () => {
    await expect(
      prepareMaskingExemptionDraft(
        { ...exact(), allowUnmatchedTarget: "true" },
        context
      )
    ).rejects.toThrow(/acknowledgment.*boolean/i);
    expect(mocks.metadata).not.toHaveBeenCalled();
  });

  it("rejects an explicit empty schema restriction rather than dropping it from the selector", async () => {
    expressions.set("empty-schema", equals("schema_name", ""));

    await expect(
      prepareMaskingExemptionDraft(
        { ...exact(), condition: "empty-schema" },
        context
      )
    ).rejects.toThrow(/cannot be represented.*selector/i);
    expect(mocks.metadata).not.toHaveBeenCalled();
  });

  it("rejects an editor revision changed during target checking", async () => {
    mocks.metadata.mockImplementation(async () => {
      mocks.snapshot.mockResolvedValue({
        project,
        sessionId: "new-editor",
        revision: "1",
      });
      return metadata();
    });

    await expect(
      prepareMaskingExemptionDraft(exact(), context)
    ).rejects.toThrow(/draft has changed/i);
  });

  it("rejects an account switch during target checking", async () => {
    mocks.metadata.mockImplementation(async () => {
      mocks.state.currentUser!.name = "users/other@example.com";
      return metadata();
    });

    await expect(
      prepareMaskingExemptionDraft(exact(), context)
    ).rejects.toThrow(/context/i);
  });

  it("does not let acknowledgment bypass the existing permission checks", async () => {
    const setup = await mocks.setup();
    setup.steps[0].configure = {
      status: "denied",
      requiredPermissions: ["bb.policies.updateMaskingExemptionPolicy"],
      missingPermissions: ["bb.policies.updateMaskingExemptionPolicy"],
      scope: project,
    };

    await expect(
      prepareMaskingExemptionDraft(
        { ...exact(), allowUnmatchedTarget: true },
        context
      )
    ).rejects.toThrow(/permission/i);
    expect(mocks.metadata).not.toHaveBeenCalled();
  });
});

it("accepts a parsed column-only predicate without table or catalog metadata and retains database scope", async () => {
  expressions.set(
    condition,
    call("_==_", [select("column_name"), literal("email")])
  );
  const { draftId, draft } = await prepareMaskingExemptionDraft(
    { ...input, expirationTimestamp: null },
    context
  );
  const group = await parseMaskingExemptionDraftCondition(draft.condition);
  expect(group.args).toEqual([
    {
      type: "Condition",
      operator: "_==_",
      args: ["resource.database", database],
    },
    {
      type: "Condition",
      operator: "_==_",
      args: ["resource.column_name", "email"],
    },
  ]);
  expect(draft.expirationTimestamp).toBeNull();
  const revalidated = await revalidateMaskingExemptionDraft(draftId, context);
  expect({ ...revalidated, condition: draft.condition }).toEqual(draft);
  expect(
    await parseMaskingExemptionDraftCondition(revalidated.condition)
  ).toEqual(group);
});

it("keeps database membership inside the confirmed database restriction", async () => {
  const names = [database, "instances/primary/databases/other"];
  expressions.set(
    "database-membership",
    call("_&&_", [
      call("@in", [
        select("database"),
        create(ExprSchema, {
          exprKind: {
            case: "listExpr",
            value: { elements: names.map(literal) },
          },
        }),
      ]),
      call("_==_", [select("column_name"), literal("email")]),
    ])
  );
  const { draft } = await prepareMaskingExemptionDraft(
    { ...input, condition: "database-membership" },
    context
  );
  const group = await parseMaskingExemptionDraftCondition(draft.condition);
  expect(group.operator).toBe("_&&_");
  expect(group.args).toEqual([
    {
      type: "Condition",
      operator: "_==_",
      args: ["resource.database", database],
    },
    { type: "Condition", operator: "@in", args: ["resource.database", names] },
    {
      type: "Condition",
      operator: "_==_",
      args: ["resource.column_name", "email"],
    },
  ]);
  for (const name of names)
    expect(mocks.database).toHaveBeenCalledWith(
      expect.objectContaining({ name }),
      expect.anything()
    );
});

it.each([
  {
    name: "column not-equal",
    parsed: call("_!=_", [select("column_name"), literal("email")]),
    operator: "_!=_",
    args: ["resource.column_name", "email"],
  },
  {
    name: "column contains",
    parsed: call("contains", [literal("mail")], select("column_name")),
    operator: "contains",
    args: ["resource.column_name", "mail"],
  },
  {
    name: "table contains",
    parsed: call("contains", [literal("user")], select("table_name")),
    operator: "contains",
    args: ["resource.table_name", "user"],
  },
  {
    name: "schema membership",
    parsed: call("@in", [
      select("schema_name"),
      create(ExprSchema, {
        exprKind: {
          case: "listExpr",
          value: { elements: [literal("public")] },
        },
      }),
    ]),
    operator: "@in",
    args: ["resource.schema_name", ["public"]],
  },
])(
  "accepts $name while retaining exact database scope",
  async ({ parsed, operator, args }) => {
    expressions.set("supported-selector", parsed);
    const predicate = { type: "Condition", operator, args };
    expect(
      await parseMaskingExemptionDraftCondition("supported-selector")
    ).toEqual({
      type: "ConditionGroup",
      operator: "_&&_",
      args: [predicate],
    });

    const { draft } = await prepareMaskingExemptionDraft(
      { ...input, condition: "supported-selector" },
      context
    );
    expect(await parseMaskingExemptionDraftCondition(draft.condition)).toEqual({
      type: "ConditionGroup",
      operator: "_&&_",
      args: [
        {
          type: "Condition",
          operator: "_==_",
          args: ["resource.database", database],
        },
        predicate,
      ],
    });
  }
);

it.each([
  ["database not-equal", call("_!=_", [select("database"), literal(database)])],
  ["database contains", call("contains", [literal("app")], select("database"))],
  [
    "computed string operand",
    call("contains", [select("column_name")], select("table_name")),
  ],
])(
  "rejects %s outside the exemption selector capabilities",
  async (_, parsed) => {
    expressions.set("unsupported-selector", parsed);
    await expect(
      parseMaskingExemptionDraftCondition("unsupported-selector")
    ).rejects.toThrow(/selector/i);
  }
);

it("resolves a verified current user and independently ANDs exact database scope", async () => {
  const { draftId, draft } = await prepareMaskingExemptionDraft(input, context);
  expect(draft.members).toEqual(["user:member@example.com"]);
  expect(draft.expirationTimestamp).toBe("2030-01-01T00:00:00.000Z");
  const serialized = JSON.stringify(
    expressions.get(draft.condition),
    (_, value) => (typeof value === "bigint" ? String(value) : value)
  );
  expect(serialized).toContain("resource.database");
  expect(serialized).toContain(database);
  expect(
    getMaskingExemptionDraft(draftId, { ...context, chatId: "other" })
  ).toBeUndefined();
  await revalidateMaskingExemptionDraft(draftId, context);
  expect(mocks.refresh).toHaveBeenCalled();
});

it("normalizes an unused empty database field for a project exemption scope", async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2029-01-01T00:00:00Z"));
  const { draft } = await prepareMaskingExemptionDraft(
    { ...input, scope: { kind: "project", database: "" } },
    context
  );

  expect(draft.scope).toEqual({ kind: "project" });
});

it.each([
  { scope: { kind: "project", database } },
  { scope: { kind: "project", database: null } },
  { scope: { kind: "project", database: false } },
  { scope: { kind: "project", database: " " } },
  { scope: { kind: "project", unexpected: "" } },
  { scope: { kind: "database", database: "" } },
  { project: "" },
])(
  "rejects conflicting or invalid exemption scope fields: %j",
  async (patch) => {
    await expect(
      prepareMaskingExemptionDraft({ ...input, ...patch }, context)
    ).rejects.toThrow(/scope|database|project/i);
  }
);

it("expires a proposal at thirty minutes without allowing it to revalidate", async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2040-01-01T00:00:00Z"));
  const { draftId } = await prepareMaskingExemptionDraft(
    {
      ...input,
      scope: { kind: "project" },
      condition: "true",
      expirationTimestamp: null,
    },
    context
  );

  vi.advanceTimersByTime(30 * 60 * 1000 - 1);
  expect(getMaskingExemptionDraft(draftId, context)).toBeDefined();
  vi.advanceTimersByTime(1);
  expect(getMaskingExemptionDraft(draftId, context)).toBeUndefined();
  await expect(
    revalidateMaskingExemptionDraft(draftId, context)
  ).rejects.toThrow(/expired/i);
  expect(mocks.refresh).not.toHaveBeenCalled();
});

it("evicts the oldest proposal while retaining the newest hundred", async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2040-01-01T00:00:00Z"));
  const proposal = {
    ...input,
    scope: { kind: "project" },
    condition: "true",
    expirationTimestamp: null,
  };
  const first = await prepareMaskingExemptionDraft(proposal, context);
  expect(getMaskingExemptionDraft(first.draftId, context)).toBeDefined();
  const ids: string[] = [];
  for (let index = 0; index < 100; index++) {
    ids.push((await prepareMaskingExemptionDraft(proposal, context)).draftId);
  }

  expect(getMaskingExemptionDraft(first.draftId, context)).toBeUndefined();
  for (const id of ids) {
    expect(getMaskingExemptionDraft(id, context)).toBeDefined();
    consumeMaskingExemptionDraft(id, context);
  }
});

it.each([
  undefined,
  "",
  "2030-01-01T00:00:00",
  "2000-01-01T00:00:00Z",
  "2030-02-31T00:00:00Z",
  "2030-01-01T24:00:00Z",
])("requires an explicit future expiry or null: %s", async (expiry) => {
  await expect(
    prepareMaskingExemptionDraft(
      { ...input, expirationTimestamp: expiry },
      context
    )
  ).rejects.toThrow(/expir/i);
});
it("keeps exact database scope outside every OR branch", async () => {
  expressions.set(
    "or",
    call("_||_", [
      call("_==_", [ident("resource.column_name"), literal("email")]),
      call("_==_", [ident("resource.column_name"), literal("phone")]),
    ])
  );
  const { draft } = await prepareMaskingExemptionDraft(
    { ...input, condition: "or", expirationTimestamp: null },
    context
  );
  const built = expressions.get(draft.condition)!;
  expect(built.exprKind.case).toBe("callExpr");
  if (built.exprKind.case !== "callExpr")
    throw new Error("Expected scoped expression");
  expect(built.exprKind.value.function).toBe("_&&_");
  expect(built.exprKind.value.args[1]?.exprKind).toMatchObject({
    case: "callExpr",
    value: { function: "_||_" },
  });
  expect(draft.expirationTimestamp).toBeNull();
});
it("keeps repeated validation idempotent without trusting database matches inside OR", async () => {
  const { draft } = await prepareMaskingExemptionDraft(input, context);
  const validated = await validateMaskingExemptionDraft(draft, context);
  const countDatabaseFactors = (expression: Expr): number => {
    const kind = expression.exprKind;
    if (kind.case === "identExpr")
      return Number(kind.value.name === "resource.database");
    if (kind.case === "callExpr")
      return kind.value.args.reduce(
        (count, arg) => count + countDatabaseFactors(arg),
        0
      );
    return 0;
  };
  expect(countDatabaseFactors(expressions.get(validated.condition)!)).toBe(1);
  expressions.set(
    "database-or-email",
    call("_||_", [
      call("_==_", [ident("resource.database"), literal(database)]),
      call("_==_", [ident("resource.column_name"), literal("email")]),
    ])
  );
  const scopedOr = await prepareMaskingExemptionDraft(
    { ...input, condition: "database-or-email" },
    context
  );
  expect(countDatabaseFactors(expressions.get(scopedOr.draft.condition)!)).toBe(
    2
  );
  expect(expressions.get(scopedOr.draft.condition)!.exprKind).toMatchObject({
    case: "callExpr",
    value: { function: "_&&_" },
  });
});
it("supports explicit all-column intent only within the confirmed project or database scope", async () => {
  const databaseDraft = await prepareMaskingExemptionDraft(
    { ...input, condition: "true" },
    context
  );
  const expression = expressions.get(databaseDraft.draft.condition)!;
  expect(expression.exprKind).toMatchObject({
    case: "callExpr",
    value: {
      function: "_==_",
      args: [
        {
          exprKind: { case: "identExpr", value: { name: "resource.database" } },
        },
        {
          exprKind: {
            case: "constExpr",
            value: { constantKind: { case: "stringValue", value: database } },
          },
        },
      ],
    },
  });
  const projectDraft = await prepareMaskingExemptionDraft(
    {
      ...input,
      scope: { kind: "project" },
      condition: "true",
      expirationTimestamp: null,
    },
    context
  );
  expect(projectDraft.draft.condition).toBe("true");
  expect(projectDraft.draft.scope).toEqual({ kind: "project" });
});
it("bounds explicitly named users to twenty", async () => {
  await expect(
    prepareMaskingExemptionDraft(
      { ...input, members: Array(21).fill("self") },
      context
    )
  ).rejects.toThrow(/member/i);
});

it("copies bound source and draft values and consumes only the owning chat's draft", async () => {
  const source = {
    sessionId: "session",
    revision: "1",
    project,
    radioValue: "ALL" as const,
    databaseResources: [],
    condition: "",
    members: [],
    description: "",
    expirationTimestamp: null,
  };
  mocks.snapshot.mockResolvedValue(source);
  const { draftId, draft } = await prepareMaskingExemptionDraft(
    input,
    context,
    { requireSourceMatch: true, source }
  );
  draft.members.push("user:unexpected@example.com");
  expect(getMaskingExemptionDraft(draftId, context)?.members).toEqual([
    "user:member@example.com",
  ]);
  const retrieved = getMaskingExemptionDraftSource(draftId, context)!;
  retrieved.revision = "changed";
  expect(getMaskingExemptionDraftSource(draftId, context)?.revision).toBe("1");
  consumeMaskingExemptionDraft(draftId, { ...context, chatId: "other" });
  expect(getMaskingExemptionDraft(draftId, context)).toBeTruthy();
  consumeMaskingExemptionDraft(draftId, context);
  expect(getMaskingExemptionDraft(draftId, context)).toBeUndefined();
});

it("rejects a source revision or identity changed during validation", async () => {
  mocks.snapshot
    .mockResolvedValueOnce(undefined)
    .mockResolvedValueOnce({ revision: "new" });
  await expect(prepareMaskingExemptionDraft(input, context)).rejects.toThrow(
    /changed/i
  );
  mocks.snapshot.mockResolvedValue(undefined);
  mocks.setup.mockImplementation(async () => {
    mocks.state = {
      ...mocks.state,
      currentUser: {
        ...mocks.state.currentUser!,
        name: "users/other@example.com",
      },
    };
    return { features: { masking: "available" }, steps: [] };
  });
  await expect(prepareMaskingExemptionDraft(input, context)).rejects.toThrow(
    /context/i
  );
});

it("fails closed on subscription loss when revalidating the click", async () => {
  const { draftId } = await prepareMaskingExemptionDraft(input, context);
  mocks.refresh.mockResolvedValue(undefined);
  await expect(
    revalidateMaskingExemptionDraft(draftId, context)
  ).rejects.toThrow(/subscription/i);
});
it("reports exact missing project permissions without guessing a role", async () => {
  const evidence = {
    status: "denied",
    scope: project,
    requiredPermissions: ["bb.policies.updateMaskingExemptionPolicy"],
    missingPermissions: ["bb.policies.updateMaskingExemptionPolicy"],
  };
  mocks.setup.mockResolvedValue({
    features: { masking: "available" },
    steps: [
      {
        id: "masking-exemptions",
        readAccess: { ...evidence, status: "allowed", missingPermissions: [] },
        configure: evidence,
      },
    ],
  });
  await expect(prepareMaskingExemptionDraft(input, context)).rejects.toThrow(
    new RegExp(
      `Scope: ${project}.*Missing permissions: bb.policies.updateMaskingExemptionPolicy`
    )
  );
});
it.each(
  [["allUsers"], ["group:team@example.com"], [], ["users/unknown"]].map(
    (members) => ({ members })
  )
)("rejects unverified or unsupported members %j", async ({ members }) => {
  await expect(
    prepareMaskingExemptionDraft({ ...input, members }, context)
  ).rejects.toThrow(/member|user/i);
});
it("resolves self and exact numeric user names through verified emails", async () => {
  const numericContext = { ...context, userName: "users/101" };
  mocks.state = {
    ...mocks.state,
    currentUser: { ...mocks.state.currentUser!, name: numericContext.userName },
  };
  mocks.user.mockResolvedValue({
    name: "users/member@example.com",
    email: "member@example.com",
    state: State.ACTIVE,
  });
  const { draft } = await prepareMaskingExemptionDraft(
    { ...input, members: ["self", "users/101"] },
    numericContext
  );
  expect(draft.members).toEqual(["user:member@example.com"]);
});
it("rejects inactive identities, wrong database ownership, and denied permission", async () => {
  mocks.user.mockResolvedValue({
    name: context.userName,
    email: "member@example.com",
    state: State.DELETED,
  });
  await expect(prepareMaskingExemptionDraft(input, context)).rejects.toThrow(
    /user|member/i
  );
  mocks.user.mockResolvedValue({
    name: context.userName,
    email: "member@example.com",
    state: State.ACTIVE,
  });
  mocks.database.mockResolvedValue({
    name: database,
    project: "projects/other",
    state: State.ACTIVE,
  });
  await expect(prepareMaskingExemptionDraft(input, context)).rejects.toThrow(
    /database/i
  );
  mocks.setup.mockResolvedValue({
    features: { masking: "available" },
    steps: [],
  });
  await expect(prepareMaskingExemptionDraft(input, context)).rejects.toThrow(
    /permission/i
  );
});
it("rejects numeric coercion and unsupported global-rule factors", async () => {
  expressions.set(
    "numeric",
    call("_==_", [
      ident("resource.column_name"),
      create(ExprSchema, {
        exprKind: {
          case: "constExpr",
          value: { constantKind: { case: "int64Value", value: 1n } },
        },
      }),
    ])
  );
  expressions.set(
    "project",
    call("_==_", [ident("resource.project_id"), literal("example")])
  );
  await expect(
    prepareMaskingExemptionDraft({ ...input, condition: "numeric" }, context)
  ).rejects.toThrow(/selector/i);
  await expect(
    prepareMaskingExemptionDraft({ ...input, condition: "project" }, context)
  ).rejects.toThrow(/selector/i);
});
