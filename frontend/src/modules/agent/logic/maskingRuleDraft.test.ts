// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AppStoreState } from "@/stores/app/types";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { PolicySchema } from "@/types/proto-es/v1/org_policy_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import {
  PlanType,
  SubscriptionSchema,
} from "@/types/proto-es/v1/subscription_service_pb";
import {
  consumeMaskingRuleDraft,
  getMaskingRuleDraft,
  getMaskingRuleDraftSource,
  getMaskingRuleDrafts,
  type MaskingRuleDraft,
  parseMaskingRuleDraftCondition,
  prepareMaskingRuleDraft,
  prepareMaskingRuleDrafts,
  revalidateMaskingRuleDraft,
  revalidateMaskingRuleDrafts,
  validateMaskingRuleDraft,
} from "./maskingRuleDraft";
import type { MaskingRuleEditorSnapshot } from "./maskingRuleDraftTarget";
import type { getMaskingSetup } from "./tools/maskingSetup";

const mocks = vi.hoisted(() => ({
  state: undefined as unknown as AppStoreState,
  getMaskingSetup: vi.fn<typeof getMaskingSetup>(),
  getSetting: vi.fn(),
  getPolicy: vi.fn(),
  getProject: vi.fn(),
  getDatabase: vi.fn(),
  batchParse: vi.fn(),
  batchDeparse: vi.fn(),
  getSnapshot: vi.fn(),
  refreshSubscription: vi.fn<AppStoreState["refreshSubscription"]>(),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => mocks.state },
}));
vi.mock("@/api", () => ({
  settingServiceClientConnect: { getSetting: mocks.getSetting },
  orgPolicyServiceClientConnect: { getPolicy: mocks.getPolicy },
  projectServiceClientConnect: { getProject: mocks.getProject },
  databaseServiceClientConnect: { getDatabase: mocks.getDatabase },
  celServiceClientConnect: {
    batchParse: mocks.batchParse,
    batchDeparse: mocks.batchDeparse,
  },
}));
vi.mock("./tools/maskingSetup", () => ({
  getMaskingSetup: mocks.getMaskingSetup,
}));
vi.mock("./maskingRuleDraftTarget", () => ({
  getMaskingRuleEditorSnapshot: mocks.getSnapshot,
}));

const context = {
  workspaceName: "workspaces/example",
  userName: "users/member@example.com",
  chatId: "masking-chat",
};
const project = "projects/example";
const database = "instances/primary/databases/app.v2";
const expression = 'resource.column_name == "email"';
const input: MaskingRuleDraft = {
  title: "Mask email",
  condition: expression,
  semanticType: "example.email",
  scope: { kind: "workspace" },
};

const ident = (name: string) =>
  create(ExprSchema, { exprKind: { case: "identExpr", value: { name } } });
const string = (value: string) =>
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
const equals = (factor: string, value: string) =>
  call("_==_", [ident(factor), string(value)]);
const expressions = new Map<string, Expr>();

function allowedSetup(): Awaited<ReturnType<typeof getMaskingSetup>> {
  const maskingRequirement = {
    feature: "FEATURE_DATA_MASKING",
    featureTitle: "Data masking",
    minimumPlan: "ENTERPRISE",
    minimumPlanTitle: "Enterprise",
  };
  const evidence = {
    status: "allowed" as const,
    scope: context.workspaceName,
    requiredPermissions: [],
    missingPermissions: [],
  };
  return {
    source: "frontend-permission-check",
    justInTime: {
      enabled: false,
      feature: "unavailable",
      featureRequirement: {
        ...maskingRequirement,
        feature: "FEATURE_JIT",
        featureTitle: "Just-in-Time Access",
      },
      create: evidence,
      eligible: false,
      exemptionFallback: false,
    },
    subscription: {
      status: "available",
      plan: "ENTERPRISE",
      planTitle: "Enterprise",
      expired: false,
    },
    featureRequirements: {
      masking: maskingRequirement,
      classification: {
        ...maskingRequirement,
        feature: "FEATURE_DATA_CLASSIFICATION",
        featureTitle: "Data classification",
      },
      catalogMasking: maskingRequirement,
    },
    features: {
      masking: "available",
      classification: "available",
      catalogMasking: "unknown",
    },
    steps: [
      {
        id: "semantic-types",
        readAccess: evidence,
        configure: evidence,
      },
      {
        id: "global-masking",
        readAccess: evidence,
        configure: evidence,
        additionalRequirements: [
          {
            ...evidence,
            source: "backend",
            permission: "bb.policies.createMaskingRulePolicy",
            condition: "when-policy-missing",
          },
        ],
      },
    ],
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-11-06T00:00:00Z"));
  mocks.state = {
    currentUser: { name: context.userName },
    serverInfo: { workspace: context.workspaceName },
    subscription: create(SubscriptionSchema, { plan: PlanType.ENTERPRISE }),
    refreshSubscription: mocks.refreshSubscription,
  } as unknown as AppStoreState;
  mocks.refreshSubscription.mockImplementation(
    async () => mocks.state.subscription
  );
  mocks.getMaskingSetup.mockResolvedValue(allowedSetup());
  mocks.getSetting.mockResolvedValue(
    create(SettingSchema, {
      name: "settings/SEMANTIC_TYPES",
      value: {
        value: {
          case: "semanticType",
          value: { types: [{ id: "example.email", title: "Email" }] },
        },
      },
    })
  );
  mocks.getPolicy.mockResolvedValue(
    create(PolicySchema, {
      policy: { case: "maskingRulePolicy", value: { rules: [] } },
    })
  );
  mocks.getProject.mockResolvedValue(
    create(ProjectSchema, { name: project, state: State.ACTIVE })
  );
  mocks.getDatabase.mockResolvedValue(
    create(DatabaseSchema$, {
      name: database,
      project,
      state: State.ACTIVE,
    })
  );
  mocks.getSnapshot.mockResolvedValue(undefined);
  expressions.clear();
  expressions.set(expression, equals("resource.column_name", "email"));
  mocks.batchParse.mockImplementation(async ({ expressions: values }) => ({
    expressions: values.map((value: string) => {
      const parsed = expressions.get(value);
      if (!parsed) throw new Error(`Unexpected CEL fixture: ${value}`);
      return parsed;
    }),
  }));
  mocks.batchDeparse.mockImplementation(async ({ expressions: values }) => ({
    expressions: values.map((value: Expr) => {
      const text = `converted-${expressions.size}`;
      expressions.set(text, value);
      return text;
    }),
  }));
});

afterEach(() => {
  vi.useRealTimers();
});

describe("masking rule drafts", () => {
  it("prepares a built-in rule before the semantic types setting exists", async () => {
    mocks.getSetting.mockRejectedValue(
      new ConnectError("missing", Code.NotFound)
    );

    const { draft } = await prepareMaskingRuleDraft(
      { ...input, semanticType: "bb.default" },
      context
    );

    expect(draft).toMatchObject({
      title: "Mask email",
      semanticType: "bb.default",
      scope: { kind: "workspace" },
    });
  });

  it("prepares full-email and partial-name rules together before the semantic types setting exists", async () => {
    mocks.getSetting.mockRejectedValue(
      new ConnectError("missing", Code.NotFound)
    );
    const nameCondition = 'resource.column_name == "name"';
    expressions.set(nameCondition, equals("resource.column_name", "name"));

    const { draftId, drafts } = await prepareMaskingRuleDrafts(
      {
        rules: [
          { ...input, semanticType: "bb.default" },
          {
            ...input,
            title: "Partially mask name",
            condition: nameCondition,
            semanticType: "bb.default-partial",
          },
        ],
      },
      context
    );

    expect(drafts.map((draft) => draft.semanticType)).toEqual([
      "bb.default",
      "bb.default-partial",
    ]);
    expect(getMaskingRuleDrafts(draftId, context)).toEqual(drafts);
    expect(
      await parseMaskingRuleDraftCondition(drafts[0].condition)
    ).toMatchObject({
      args: [{ args: ["resource.column_name", "email"] }],
    });
    expect(
      await parseMaskingRuleDraftCondition(drafts[1].condition)
    ).toMatchObject({
      args: [{ args: ["resource.column_name", "name"] }],
    });
  });

  it("prepares all proposals atomically under one identity-bound, one-use ID", async () => {
    const { draftId, drafts } = await prepareMaskingRuleDrafts(
      { rules: [input, { ...input, title: "Second rule" }] },
      context
    );
    expect(drafts).toHaveLength(2);
    expect(getMaskingRuleDrafts(draftId, context)).toEqual(drafts);
    expect(getMaskingRuleDraft(draftId, context)).toEqual(drafts[0]);
    const revalidated = await revalidateMaskingRuleDrafts(draftId, context);
    expect(revalidated).toHaveLength(2);
    expect(getMaskingRuleDrafts(draftId, context)).toEqual(revalidated);
    expect(getMaskingRuleDraft(draftId, context)).toEqual(revalidated[0]);
    revalidated[0].title = "Mutated outside registry";
    expect(getMaskingRuleDraft(draftId, context)?.title).toBe("Mask email");
    expect(
      getMaskingRuleDrafts(draftId, { ...context, chatId: "other" })
    ).toBeUndefined();
    consumeMaskingRuleDraft(draftId, context);
    expect(getMaskingRuleDrafts(draftId, context)).toBeUndefined();
  });

  it("normalizes each batch scope before comparing confirmed scopes", async () => {
    const { drafts } = await prepareMaskingRuleDrafts(
      {
        rules: [
          {
            ...input,
            scope: { kind: "workspace", project: "", database: "" },
          },
          { ...input, title: "Second rule" },
        ],
      },
      context
    );

    expect(drafts.map((draft) => draft.scope)).toEqual([
      { kind: "workspace" },
      { kind: "workspace" },
    ]);
  });

  it("provides consistent single and batch reads when revalidating a singleton", async () => {
    const { draftId, draft } = await prepareMaskingRuleDraft(input, context);
    expect(getMaskingRuleDrafts(draftId, context)).toEqual([draft]);
    const revalidated = await revalidateMaskingRuleDraft(draftId, context);
    expect(getMaskingRuleDraft(draftId, context)).toEqual(revalidated);
    expect(getMaskingRuleDrafts(draftId, context)).toEqual([revalidated]);
    expect(await revalidateMaskingRuleDrafts(draftId, context)).toHaveLength(1);
  });

  it("allows reviewing a removed semantic type only so the member can replace it", async () => {
    const proposal = { ...input, semanticType: "removed" };
    await expect(validateMaskingRuleDraft(proposal, context)).rejects.toThrow(
      /semantic type/i
    );
    await expect(
      validateMaskingRuleDraft(proposal, context, { purpose: "review" })
    ).resolves.toMatchObject({ semanticType: "removed" });
    const setup = allowedSetup();
    setup.steps[1].configure = {
      ...setup.steps[1].configure,
      status: "denied",
    };
    mocks.getMaskingSetup.mockResolvedValue(setup);
    await expect(
      validateMaskingRuleDraft(proposal, context, { purpose: "review" })
    ).rejects.toThrow(/permission/i);
  });

  it("does not duplicate scoped equality conjuncts when queue drafts are repeatedly validated", async () => {
    let draft = await validateMaskingRuleDraft(
      { ...input, scope: { kind: "database", database, project } },
      context
    );
    const original = await parseMaskingRuleDraftCondition(draft.condition);
    for (let index = 0; index < 3; index++) {
      draft = await validateMaskingRuleDraft(draft, context, {
        purpose: "review",
      });
      expect(await parseMaskingRuleDraftCondition(draft.condition)).toEqual(
        original
      );
    }
  });

  it("keeps an independent project restriction when the same equality appears only inside OR", async () => {
    const condition =
      'resource.project_id == "example" || resource.column_name == "email"';
    expressions.set(
      condition,
      call("_||_", [
        equals("resource.project_id", "example"),
        equals("resource.column_name", "email"),
      ])
    );
    const draft = await validateMaskingRuleDraft(
      { ...input, condition, scope: { kind: "project", project } },
      context
    );
    const parsed = await parseMaskingRuleDraftCondition(draft.condition);
    expect(parsed).toMatchObject({
      operator: "_&&_",
      args: [
        {
          type: "Condition",
          operator: "_==_",
          args: ["resource.project_id", "example"],
        },
        { type: "ConditionGroup", operator: "_||_" },
      ],
    });
    const repeated = await validateMaskingRuleDraft(draft, context);
    expect(await parseMaskingRuleDraftCondition(repeated.condition)).toEqual(
      parsed
    );
  });

  it.each(
    [
      [],
      [input],
      Array.from({ length: 11 }, () => input),
      [input, { ...input, semanticType: "missing" }],
      [input, { ...input, scope: { kind: "project", project } }],
    ].map((rules) => [rules])
  )("rejects an invalid batch without partial delivery: %j", async (rules) => {
    await expect(
      prepareMaskingRuleDrafts({ rules }, context)
    ).rejects.toThrow();
  });

  it("does not prepare a batch over an active editor", async () => {
    mocks.getSnapshot.mockResolvedValue({ sessionId: "active", revision: "1" });
    await expect(
      prepareMaskingRuleDrafts({ rules: [input, input] }, context)
    ).rejects.toThrow(/editor/i);
    expect(mocks.getMaskingSetup).not.toHaveBeenCalled();
  });
  it("prepares a bounded unsaved rule with observed semantic type", async () => {
    const { draftId, draft } = await prepareMaskingRuleDraft(input, context);

    expect(draftId).toBeTruthy();
    expect(draft).toMatchObject({
      title: "Mask email",
      semanticType: "example.email",
      scope: { kind: "workspace" },
    });
    expect(getMaskingRuleDraft(draftId, context)).toEqual(draft);
    draft.title = "Mutated outside registry";
    expect(getMaskingRuleDraft(draftId, context)?.title).toBe("Mask email");
  });

  it.each([
    {
      scope: { kind: "workspace", project: "", database: "" },
      expected: { kind: "workspace" },
    },
    {
      scope: { kind: "project", project, database: "" },
      expected: { kind: "project", project },
    },
  ])(
    "normalizes empty optional scope fields for $scope.kind rules",
    async ({ scope, expected }) => {
      const { draft } = await prepareMaskingRuleDraft(
        { ...input, scope },
        context
      );

      expect(draft.scope).toEqual(expected);
    }
  );

  it.each([
    { title: "" },
    { title: "x".repeat(201) },
    { condition: "x".repeat(20_001) },
    { condition: "resource.column_name\0" },
    { semanticType: "x".repeat(201) },
    { scope: { kind: "project", project: "projects/-" } },
    { scope: { kind: "project", project: "" } },
    { scope: { kind: "database", project: "", database } },
    { scope: { kind: "database", project, database: "" } },
  ])("rejects malformed or oversized proposal fields: %j", async (patch) => {
    await expect(
      prepareMaskingRuleDraft({ ...input, ...patch }, context)
    ).rejects.toThrow();
  });

  it.each([
    { ...input, unexpected: true },
    { ...input, scope: { kind: "workspace", project } },
    { ...input, scope: { kind: "project", project, database } },
    { ...input, scope: { kind: "database", project, database, mode: "mask" } },
    { ...input, scope: { kind: "workspace", project: null } },
    { ...input, scope: { kind: "workspace", database: false } },
    { ...input, scope: { kind: "workspace", project: " " } },
    { ...input, scope: { kind: "project", project, database: null } },
    { ...input, scope: { kind: "project", project, database: false } },
    { ...input, scope: { kind: "project", project, database: " " } },
    { ...input, scope: { kind: "workspace", unexpected: "" } },
  ])(
    "rejects unexpected draft or scope fields rather than widening scope: %j",
    async (raw) => {
      await expect(prepareMaskingRuleDraft(raw, context)).rejects.toThrow(
        /unexpected/i
      );
    }
  );

  it("rejects lowerAscii rather than silently falling back to raw CEL", async () => {
    const condition = 'resource.column_name.lowerAscii() == "email"';
    expressions.set(
      condition,
      call("_==_", [
        call("lowerAscii", [], ident("resource.column_name")),
        string("email"),
      ])
    );

    await expect(
      prepareMaskingRuleDraft({ ...input, condition }, context)
    ).rejects.toThrow(/selector/i);
  });

  it("rejects a type coercion that would change the proposed condition", async () => {
    const condition = 'resource.classification_level == "confidential"';
    expressions.set(
      condition,
      equals("resource.classification_level", "confidential")
    );

    await expect(parseMaskingRuleDraftCondition(condition)).rejects.toThrow();
  });

  it("rejects a conversion that drops the independently constructed scope", async () => {
    mocks.batchDeparse.mockImplementation(async ({ expressions: values }) => ({
      expressions: values.map(() => expression),
    }));

    await expect(
      prepareMaskingRuleDraft(
        { ...input, scope: { kind: "project", project } },
        context
      )
    ).rejects.toThrow(/conversion/i);
  });

  it("rejects a changed CEL roundtrip", async () => {
    const changed = 'resource.column_name == "other"';
    expressions.set(changed, equals("resource.column_name", "other"));
    mocks.batchDeparse.mockResolvedValue({ expressions: [changed] });

    await expect(parseMaskingRuleDraftCondition(expression)).rejects.toThrow(
      /conversion/i
    );
  });

  it.each(["global-masking", "semantic-types"])(
    "requires verified %s read permission",
    async (id) => {
      const setup = allowedSetup();
      const step = setup.steps.find((item) => item.id === id)!;
      step.readAccess = { ...step.readAccess, status: "unknown" };
      mocks.getMaskingSetup.mockResolvedValue(setup);

      await expect(prepareMaskingRuleDraft(input, context)).rejects.toThrow(
        /permission/i
      );
    }
  );

  it("does not accept a builtin semantic type when settings cannot be observed", async () => {
    mocks.getSetting.mockRejectedValue(new Error("Settings unavailable"));
    await expect(
      prepareMaskingRuleDraft({ ...input, semanticType: "bb.default" }, context)
    ).rejects.toThrow();
  });

  it("does not accept a raw expression when parsing fails", async () => {
    mocks.batchParse.mockResolvedValue({ expressions: [create(ExprSchema)] });
    await expect(prepareMaskingRuleDraft(input, context)).rejects.toThrow(
      /parsed/i
    );
  });

  it.each(["denied", "unknown"] as const)(
    "rejects %s configuration permission evidence",
    async (status) => {
      const setup = allowedSetup();
      setup.steps[1].configure = { ...setup.steps[1].configure, status };
      mocks.getMaskingSetup.mockResolvedValue(setup);

      await expect(prepareMaskingRuleDraft(input, context)).rejects.toThrow(
        /permission/i
      );
    }
  );

  it.each([
    [1, "readAccess", "bb.policies.getMaskingRulePolicy", "denied"],
    [1, "readAccess", "bb.policies.getMaskingRulePolicy", "unknown"],
    [1, "configure", "bb.policies.updateMaskingRulePolicy", "denied"],
    [1, "configure", "bb.policies.updateMaskingRulePolicy", "unknown"],
    [0, "readAccess", "bb.settings.get", "denied"],
    [0, "readAccess", "bb.settings.get", "unknown"],
  ] as const)(
    "identifies scope, required and missing permissions for %s %s %s %s",
    async (index, access, permission, status) => {
      const setup = allowedSetup();
      setup.steps[index][access] = {
        ...setup.steps[index][access],
        requiredPermissions: [permission],
        missingPermissions: status === "denied" ? [permission] : [],
        status,
      };
      mocks.getMaskingSetup.mockResolvedValue(setup);

      const result = prepareMaskingRuleDraft(input, context);
      await expect(result).rejects.toThrow(`Permission status: ${status}`);
      await expect(result).rejects.toThrow(`Scope: ${context.workspaceName}`);
      await expect(result).rejects.toThrow(
        `Required permissions: ${permission}`
      );
      await expect(result).rejects.toThrow(
        `Missing permissions: ${status === "denied" ? permission : "unverified"}`
      );
    }
  );

  it.each(["unavailable", "unknown"] as const)(
    "rejects %s masking feature evidence",
    async (status) => {
      const setup = allowedSetup();
      setup.features.masking = status;
      mocks.getMaskingSetup.mockResolvedValue(setup);

      await expect(prepareMaskingRuleDraft(input, context)).rejects.toThrow(
        /feature/i
      );
    }
  );

  it("rejects a semantic type absent from fresh settings", async () => {
    await expect(
      prepareMaskingRuleDraft(
        { ...input, semanticType: "example.missing" },
        context
      )
    ).rejects.toThrow(/semantic type/i);
  });

  it.each(["denied", "unknown"] as const)(
    "identifies %s create permission when the masking policy does not exist",
    async (status) => {
      mocks.getPolicy.mockRejectedValue(
        new ConnectError("missing", Code.NotFound)
      );
      const setup = allowedSetup();
      const requirement = setup.steps[1].additionalRequirements![0];
      if (requirement.source !== "backend") throw new Error("fixture");
      requirement.status = status;
      requirement.requiredPermissions = ["bb.policies.createMaskingRulePolicy"];
      requirement.missingPermissions =
        status === "denied" ? ["bb.policies.createMaskingRulePolicy"] : [];
      mocks.getMaskingSetup.mockResolvedValue(setup);

      const result = prepareMaskingRuleDraft(input, context);
      await expect(result).rejects.toThrow(`Permission status: ${status}`);
      await expect(result).rejects.toThrow(`Scope: ${context.workspaceName}`);
      await expect(result).rejects.toThrow(
        "Required permissions: bb.policies.createMaskingRulePolicy"
      );
    }
  );

  it("conjoins database scope outside an OR and does not duplicate it on refresh", async () => {
    const condition =
      'resource.column_name == "email" || resource.project_id == "other"';
    expressions.set(
      condition,
      call("_||_", [
        equals("resource.column_name", "email"),
        equals("resource.project_id", "other"),
      ])
    );
    const { draftId, draft } = await prepareMaskingRuleDraft(
      { ...input, condition, scope: { kind: "database", database, project } },
      context
    );
    const parsed = await parseMaskingRuleDraftCondition(draft.condition);

    expect(parsed).toEqual({
      type: "ConditionGroup",
      operator: "_&&_",
      args: [
        {
          type: "Condition",
          operator: "_==_",
          args: ["resource.project_id", "example"],
        },
        {
          type: "Condition",
          operator: "_==_",
          args: ["resource.instance_id", "primary"],
        },
        {
          type: "Condition",
          operator: "_==_",
          args: ["resource.database_name", "app.v2"],
        },
        {
          type: "ConditionGroup",
          operator: "_||_",
          args: [
            {
              type: "Condition",
              operator: "_==_",
              args: ["resource.column_name", "email"],
            },
            {
              type: "Condition",
              operator: "_==_",
              args: ["resource.project_id", "other"],
            },
          ],
        },
      ],
    });
    const refreshed = await revalidateMaskingRuleDraft(draftId, context);
    expect(await parseMaskingRuleDraftCondition(refreshed.condition)).toEqual(
      parsed
    );
  });

  it.each(["other-project", "deleted-database", "deleted-project"])(
    "rejects fresh scope metadata with %s",
    async (failure) => {
      if (failure === "deleted-project")
        mocks.getProject.mockResolvedValue(
          create(ProjectSchema, { name: project, state: State.DELETED })
        );
      else
        mocks.getDatabase.mockResolvedValue(
          create(DatabaseSchema$, {
            name: database,
            project: failure === "other-project" ? "projects/other" : project,
            state:
              failure === "deleted-database" ? State.DELETED : State.ACTIVE,
          })
        );

      await expect(
        prepareMaskingRuleDraft(
          {
            ...input,
            scope: { kind: "database", database, project },
          },
          context
        )
      ).rejects.toThrow();
    }
  );

  it("revalidates fresh permission evidence instead of reusing the preparation verdict", async () => {
    const { draftId } = await prepareMaskingRuleDraft(input, context);
    const setup = allowedSetup();
    setup.steps[1].configure = {
      ...setup.steps[1].configure,
      status: "denied",
    };
    mocks.getMaskingSetup.mockResolvedValue(setup);

    await expect(revalidateMaskingRuleDraft(draftId, context)).rejects.toThrow(
      /permission/i
    );
  });

  it.each(["unavailable", "failed"])(
    "does not reuse cached entitlement when click-time subscription refresh is %s",
    async (failure) => {
      const { draftId } = await prepareMaskingRuleDraft(input, context);
      if (failure === "unavailable")
        mocks.refreshSubscription.mockResolvedValue(undefined);
      else mocks.refreshSubscription.mockRejectedValue(new Error("network"));

      await expect(
        revalidateMaskingRuleDraft(draftId, context)
      ).rejects.toThrow(/feature.*unverified/i);
      expect(getMaskingRuleDraft(draftId, context)).toBeDefined();
    }
  );

  it("rejects a feature revoked by the fresh click-time subscription", async () => {
    mocks.getMaskingSetup.mockImplementation(async () => {
      const setup = allowedSetup();
      if (mocks.state.subscription?.plan === PlanType.FREE)
        setup.features.masking = "unavailable";
      return setup;
    });
    const { draftId } = await prepareMaskingRuleDraft(input, context);
    mocks.refreshSubscription.mockImplementation(async () => {
      const subscription = create(SubscriptionSchema, { plan: PlanType.FREE });
      mocks.state = { ...mocks.state, subscription };
      return subscription;
    });

    await expect(revalidateMaskingRuleDraft(draftId, context)).rejects.toThrow(
      /feature/i
    );
  });

  it("rejects an actor change while click-time subscription refresh is pending", async () => {
    const { draftId } = await prepareMaskingRuleDraft(input, context);
    mocks.refreshSubscription.mockImplementation(async () => {
      mocks.state = {
        ...mocks.state,
        currentUser: {
          name: "users/other@example.com",
        } as AppStoreState["currentUser"],
      };
      return mocks.state.subscription;
    });

    await expect(revalidateMaskingRuleDraft(draftId, context)).rejects.toThrow(
      /context/i
    );
  });

  it("rejects a semantic type removed after preparation", async () => {
    const { draftId } = await prepareMaskingRuleDraft(input, context);
    mocks.getSetting.mockResolvedValue(
      create(SettingSchema, {
        value: { value: { case: "semanticType", value: { types: [] } } },
      })
    );
    await expect(revalidateMaskingRuleDraft(draftId, context)).rejects.toThrow(
      /semantic type/i
    );
  });

  it("rejects a database moved out of the project after preparation", async () => {
    const { draftId } = await prepareMaskingRuleDraft(
      {
        ...input,
        scope: { kind: "database", database, project },
      },
      context
    );
    mocks.getDatabase.mockResolvedValue(
      create(DatabaseSchema$, {
        name: database,
        project: "projects/other",
        state: State.ACTIVE,
      })
    );
    await expect(revalidateMaskingRuleDraft(draftId, context)).rejects.toThrow(
      /outside the project/i
    );
  });

  it("expires the registry after 30 minutes and binds it to actor, workspace, and chat", async () => {
    const { draftId } = await prepareMaskingRuleDraft(input, context);
    for (const changed of [
      { userName: "users/other@example.com" },
      { workspaceName: "workspaces/other" },
      { chatId: "other-chat" },
    ]) {
      const otherContext = { ...context, ...changed };
      expect(getMaskingRuleDraft(draftId, otherContext)).toBeUndefined();
      consumeMaskingRuleDraft(draftId, otherContext);
      expect(getMaskingRuleDraft(draftId, context)).toBeDefined();
    }
    vi.advanceTimersByTime(30 * 60 * 1000);
    expect(getMaskingRuleDraft(draftId, context)).toBeUndefined();
    await expect(
      revalidateMaskingRuleDraft(draftId, context)
    ).rejects.toThrow();
  });

  it("consumes only the matching draft", async () => {
    const { draftId } = await prepareMaskingRuleDraft(input, context);
    consumeMaskingRuleDraft(draftId, context);
    expect(getMaskingRuleDraft(draftId, context)).toBeUndefined();
  });

  it("captures an editor source once and rejects a source from another chat", async () => {
    const source: MaskingRuleEditorSnapshot = {
      sessionId: "editor-session",
      revision: "1",
      title: "Original",
      condition: expression,
      semanticType: "example.email",
      chatId: context.chatId,
    };
    mocks.getSnapshot.mockResolvedValue(source);
    const { draftId } = await prepareMaskingRuleDraft(input, context);
    mocks.getSnapshot.mockResolvedValue({ ...source, revision: "2" });
    await revalidateMaskingRuleDraft(draftId, context);
    expect(getMaskingRuleDraftSource(draftId, context)).toEqual(source);
    mocks.getSnapshot.mockResolvedValue({ ...source, chatId: "other-chat" });
    await expect(prepareMaskingRuleDraft(input, context)).rejects.toThrow(
      /chat/i
    );
  });

  it.each([undefined, "create"] as const)(
    "preserves new-rule preparation with editor mode %s",
    async (mode) => {
      mocks.getSnapshot.mockResolvedValue({
        sessionId: "new-editor",
        revision: "1",
        mode,
        title: "New email rule",
        condition: expression,
        semanticType: "example.email",
      });

      const { draftId } = await prepareMaskingRuleDraft(input, context);
      expect(getMaskingRuleDraft(draftId, context)).toBeDefined();
    }
  );

  it.each([false, true])(
    "rejects preparing a full-rule draft from an existing-rule editor with source matching %s",
    async (requireSourceMatch) => {
      const source: MaskingRuleEditorSnapshot = {
        sessionId: "existing-editor",
        revision: "1",
        mode: "edit",
        title: "Existing email rule",
        condition: expression,
        semanticType: "example.email",
      };
      mocks.getSnapshot.mockResolvedValue(source);

      await expect(
        prepareMaskingRuleDraft(
          input,
          context,
          requireSourceMatch ? { requireSourceMatch: true, source } : undefined
        )
      ).rejects.toThrow(/existing.*review-only/i);
      expect(mocks.getMaskingSetup).not.toHaveBeenCalled();
      expect(mocks.batchParse).not.toHaveBeenCalled();
    }
  );

  it.each([undefined, "previous-revision", "current-revision"])(
    "rejects a current editor not matching the model's last read: %j",
    async (previousRevision) => {
      const current: MaskingRuleEditorSnapshot = {
        sessionId: "editor-session",
        revision: "current-revision",
        title: "Manually changed",
        condition: expression,
        semanticType: "example.email",
        chatId: context.chatId,
      };
      mocks.getSnapshot.mockResolvedValue(current);

      await expect(
        prepareMaskingRuleDraft(input, context, {
          requireSourceMatch: true,
          source: previousRevision
            ? { ...current, revision: previousRevision, title: "Earlier read" }
            : undefined,
        })
      ).rejects.toThrow(/Read current unsaved draft again/);
    }
  );

  it("prepares a new rule when no unsaved editor was read or is active", async () => {
    const { draftId } = await prepareMaskingRuleDraft(input, context, {
      requireSourceMatch: true,
      source: undefined,
    });

    expect(getMaskingRuleDraft(draftId, context)).toBeDefined();
    expect(getMaskingRuleDraftSource(draftId, context)).toBeUndefined();
  });

  it("anchors the proposal to the exact editor snapshot the model read", async () => {
    const source: MaskingRuleEditorSnapshot = {
      sessionId: "editor-session",
      revision: "1",
      title: "Earlier read",
      condition: expression,
      semanticType: "example.email",
      chatId: context.chatId,
    };
    mocks.getSnapshot.mockResolvedValue(source);

    const { draftId } = await prepareMaskingRuleDraft(input, context, {
      requireSourceMatch: true,
      source,
    });

    expect(getMaskingRuleDraftSource(draftId, context)).toEqual(source);
  });

  it.each([false, true])(
    "rejects editor changes during metadata validation with source matching %s",
    async (requireSourceMatch) => {
      const source: MaskingRuleEditorSnapshot = {
        sessionId: "editor-session",
        revision: "1",
        title: "Earlier read",
        condition: expression,
        semanticType: "example.email",
        chatId: context.chatId,
      };
      mocks.getSnapshot.mockResolvedValue(source);
      mocks.getSetting.mockImplementation(async () => {
        mocks.getSnapshot.mockResolvedValue({
          ...source,
          revision: "2",
          title: "Manually changed during validation",
        });
        return create(SettingSchema, {
          name: "settings/SEMANTIC_TYPES",
          value: {
            value: {
              case: "semanticType",
              value: { types: [{ id: "example.email", title: "Email" }] },
            },
          },
        });
      });

      await expect(
        prepareMaskingRuleDraft(
          input,
          context,
          requireSourceMatch ? { requireSourceMatch: true, source } : undefined
        )
      ).rejects.toThrow(/Read current unsaved draft again/);
    }
  );

  it.each([undefined, ""])(
    "only treats an undefined source chat as an unbound manual editor: %j",
    async (chatId) => {
      mocks.getSnapshot.mockResolvedValue({
        sessionId: "manual-editor",
        revision: "1",
        title: "Original",
        condition: expression,
        semanticType: "example.email",
        chatId,
      });
      const result = prepareMaskingRuleDraft(input, context);
      if (chatId === undefined) {
        const { draftId } = await result;
        expect(getMaskingRuleDraftSource(draftId, context)).toMatchObject({
          sessionId: "manual-editor",
        });
      } else await expect(result).rejects.toThrow(/chat/i);
    }
  );

  it("rejects an account switch during fresh validation", async () => {
    mocks.getSetting.mockImplementation(async () => {
      mocks.state = {
        ...mocks.state,
        currentUser: {
          name: "users/other@example.com",
        } as AppStoreState["currentUser"],
      };
      return create(SettingSchema, {
        value: {
          value: {
            case: "semanticType",
            value: { types: [{ id: "example.email" }] },
          },
        },
      });
    });
    await expect(prepareMaskingRuleDraft(input, context)).rejects.toThrow();
  });
});
