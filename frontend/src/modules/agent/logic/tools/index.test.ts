// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { AppRouterInstance } from "@/app/router";
import i18n from "@/lib/i18n";
import { useAppStore } from "@/stores/app";
import {
  ConstantSchema,
  type Expr,
  Expr_CallSchema,
  Expr_IdentSchema,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import {
  ApprovalStatus,
  IssueStatus,
  State,
} from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import {
  ApprovalFlowSchema,
  ApprovalTemplateSchema,
  Issue_Type,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { MaskingRuleEditorSnapshot } from "../maskingRuleDraftTarget";
import { getMaskingSkillGuidance } from "../skills";

describe("progressive masking tool definitions", () => {
  test("keeps masking schemas out of an unrelated Plan request", () => {
    const names = getToolDefinitions({
      referencedProject: "projects/example",
    }).map((tool) => tool.name);
    expect(names).toContain("get_plan_overview");
    expect(names).toContain("get_skill");
    expect(names).not.toContain("get_masking_overview");
    expect(names).not.toContain("prepare_masking_rule_draft");
  });

  test("advertises only loaded bundles and their overview dependency", () => {
    const tools = getToolDefinitions({ maskingSections: ["rule"] });
    const names = tools.map((tool) => tool.name);
    expect(names).toContain("get_masking_overview");
    expect(names).toContain("ask_masking_rule_scope");
    expect(names).toContain("ask_masking_semantic_type");
    expect(names).not.toContain("prepare_masking_exemption_draft");
    expect(names).not.toContain("prepare_semantic_type_draft");
    expect(new Set(names).size).toBe(names.length);
  });

  test("activates only successful skill loads and acknowledges masking content", async () => {
    const onSkillLoaded = vi.fn();
    const executor = createToolExecutor({} as AppRouterInstance, {
      onSkillLoaded,
    });
    const result = await executor(
      "get_skill",
      { name: "data-masking", section: "rule" },
      "load"
    );
    expect(result).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        kind: "loaded",
        name: "data-masking",
        section: "rule",
      }),
    });
    expect(onSkillLoaded).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        kind: "loaded",
        name: "data-masking",
        section: "rule",
      })
    );
    onSkillLoaded.mockClear();
    await executor(
      "get_skill",
      { name: "data-masking", section: "invalid" },
      "bad"
    );
    await executor("get_skill", {}, "list");
    expect(onSkillLoaded).not.toHaveBeenCalled();
  });
});

const projectApi = vi.hoisted(() => ({ getProject: vi.fn() }));
const issueApi = vi.hoisted(() => ({ getIssue: vi.fn() }));
const planApi = vi.hoisted(() => ({ getPlan: vi.fn() }));
const databaseApi = vi.hoisted(() => ({ getDatabase: vi.fn() }));
const semanticApi = vi.hoisted(() => ({
  setting: vi.fn(),
  prepare: vi.fn(),
  summary: vi.fn(),
  setup: vi.fn(),
}));
vi.mock("../semanticTypeDraft", () => ({
  prepareSemanticTypeDraft: semanticApi.prepare,
  getSafeSemanticTypeDraftSummary: semanticApi.summary,
}));
vi.mock("./maskingSetup", () => ({ getMaskingSetup: semanticApi.setup }));
const celApi = vi.hoisted(() => ({ batchParse: vi.fn() }));
const resourceChoiceApi = vi.hoisted(() => ({ validate: vi.fn() }));
const instanceOverviewApi = vi.hoisted(() => ({
  create: vi.fn(() => vi.fn(async () => "instance_overview")),
  resolveName: vi.fn((path: string) =>
    path === "/instances/primary" ? "instances/primary" : undefined
  ),
}));
const maskingOverviewApi = vi.hoisted(() => ({
  create: vi.fn(() => vi.fn(async () => "masking_overview")),
}));
const maskingDraftApi = vi.hoisted(() => ({
  prepare: vi.fn(async () => ({ draftId: "draft-1", draft: {} })),
  prepareMany: vi.fn(async () => ({ draftId: "batch-1", drafts: [{}, {}] })),
  prepareExemption: vi.fn(async () => ({ draftId: "exemption-1", draft: {} })),
  exemptionSnapshot: vi.fn(async (): Promise<unknown> => undefined),
  snapshot: vi.fn<() => Promise<MaskingRuleEditorSnapshot | undefined>>(
    async () => undefined
  ),
}));

vi.mock("../maskingRuleDraft", () => ({
  prepareMaskingRuleDraft: maskingDraftApi.prepare,
  prepareMaskingRuleDrafts: maskingDraftApi.prepareMany,
}));
vi.mock("../maskingExemptionDraft", () => ({
  prepareMaskingExemptionDraft: maskingDraftApi.prepareExemption,
}));
vi.mock("../maskingExemptionDraftTarget", () => ({
  getMaskingExemptionEditorSnapshot: maskingDraftApi.exemptionSnapshot,
}));
vi.mock("../maskingRuleDraftTarget", () => ({
  getMaskingRuleEditorSnapshot: maskingDraftApi.snapshot,
}));

vi.mock("@/api", () => ({
  projectServiceClientConnect: { getProject: projectApi.getProject },
  issueServiceClientConnect: { getIssue: issueApi.getIssue },
  planServiceClientConnect: { getPlan: planApi.getPlan },
  databaseServiceClientConnect: { getDatabase: databaseApi.getDatabase },
  settingServiceClientConnect: { getSetting: semanticApi.setting },
  celServiceClientConnect: { batchParse: celApi.batchParse },
}));

vi.mock("../skills", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../skills")>()),
  getSkill: vi.fn(async () => "skill"),
}));
vi.mock("./callApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./callApi")>()),
  dispatchPreparedApiCall: vi.fn(async () => "call_api"),
}));
vi.mock("./apiPermissionPreflight", () => ({
  checkPreparedApiPermission: vi.fn(async () => ({ kind: "allowed" })),
}));
vi.mock("./resourceChoices", () => ({
  validateResourceChoices: resourceChoiceApi.validate,
}));
vi.mock("./databaseCount", () => ({
  createDatabaseCountTool: vi.fn(() => vi.fn(async () => "database_count")),
}));
vi.mock("./databaseOverview", () => ({
  createDatabaseOverviewTool: vi.fn(() =>
    vi.fn(async () => "database_overview")
  ),
}));
vi.mock("./instanceOverview", () => ({
  createInstanceOverviewTool: instanceOverviewApi.create,
  resolveInstanceNameFromPagePath: instanceOverviewApi.resolveName,
}));
vi.mock("./maskingOverview", () => ({
  createMaskingOverviewTool: maskingOverviewApi.create,
}));
vi.mock("./domAction", () => ({
  createDomActionTool: vi.fn(() => vi.fn(async () => "dom_action")),
}));
vi.mock("./navigate", () => ({
  createNavigateTool: vi.fn(() => vi.fn(async () => "navigate")),
}));
vi.mock("./pageState", () => ({
  createPageStateTool: vi.fn(() => vi.fn(async () => "page_state")),
}));
vi.mock("./projectOpenIssues", () => ({
  createProjectOpenIssuesTool: vi.fn(() => vi.fn(async () => "project_issues")),
}));
vi.mock("./projectOverview", () => ({
  createProjectOverviewTool: vi.fn(() => vi.fn(async () => "project_overview")),
}));
vi.mock("./projectGovernanceOverview", () => ({
  createProjectGovernanceOverviewTool: vi.fn(() =>
    vi.fn(async () => "project_governance_overview")
  ),
}));
vi.mock("./issueOverview", () => ({
  createIssueOverviewTool: vi.fn(() => vi.fn(async () => "issue_overview")),
}));
vi.mock("./planOverview", () => ({
  createPlanOverviewTool: vi.fn(() => vi.fn(async () => "plan_overview")),
}));
vi.mock("./workspaceOverview", () => ({
  createWorkspaceOverviewTool: vi.fn(() =>
    vi.fn(async () => "workspace_overview")
  ),
}));
vi.mock("./projectRecentActivity", () => ({
  createProjectRecentActivityTool: vi.fn(() =>
    vi.fn(async () => "project_recent_activity")
  ),
}));
vi.mock("./workspaceRecentActivity", () => ({
  createWorkspaceRecentActivityTool: vi.fn(() =>
    vi.fn(async () => "workspace_recent_activity")
  ),
}));
vi.mock("./workspaceAttention", () => ({
  createWorkspaceAttentionTool: vi.fn(() =>
    vi.fn(async () => "workspace_attention")
  ),
}));
vi.mock("./searchApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./searchApi")>()),
  searchApi: vi.fn(async () => "search_api"),
}));

import { createToolExecutor, getToolDefinitions } from ".";
import { checkPreparedApiPermission } from "./apiPermissionPreflight";
import { dispatchPreparedApiCall } from "./callApi";
import { createDatabaseCountTool } from "./databaseCount";
import { createDatabaseOverviewTool } from "./databaseOverview";
import { createDomActionTool } from "./domAction";
import { createIssueOverviewTool } from "./issueOverview";
import { createNavigateTool } from "./navigate";
import { createPageStateTool } from "./pageState";
import {
  getPendingApiCallPreview,
  resolvePendingApiCall,
} from "./pendingApiCall";
import { createPlanOverviewTool } from "./planOverview";
import { createProjectGovernanceOverviewTool } from "./projectGovernanceOverview";
import { createProjectOpenIssuesTool } from "./projectOpenIssues";
import { createProjectOverviewTool } from "./projectOverview";
import { createProjectRecentActivityTool } from "./projectRecentActivity";
import { createWorkspaceAttentionTool } from "./workspaceAttention";
import { createWorkspaceOverviewTool } from "./workspaceOverview";
import { createWorkspaceRecentActivityTool } from "./workspaceRecentActivity";

test("builds semantic type choices from current settings rather than model options", async () => {
  const execute = createToolExecutor({} as AppRouterInstance, {
    workspaceName: "workspaces/example",
    userName: "users/member@example.com",
    chatId: "chat",
  });
  const result = await execute("ask_masking_semantic_type", {}, "choose-type");
  expect(result.kind).toBe("ask_user");
  if (result.kind !== "ask_user") return;
  expect(result.ask.options?.map((option) => option.value)).toEqual([
    "semantic-type:bb.default",
    "semantic-type:bb.default-partial",
    "action:create-semantic-type",
  ]);
  expect(result.ask.defaultValue).toBeUndefined();
  expect(
    (await execute("ask_masking_semantic_type", { options: [] }, "bad")).kind
  ).toBe("tool_result");
});

test("prepares semantic type drafts without exposing private algorithm fields", async () => {
  const execute = createToolExecutor({} as AppRouterInstance, {
    workspaceName: "workspaces/example",
    userName: "users/member@example.com",
    chatId: "chat",
  });
  const args = { title: "New type", algorithm: { kind: "md5-mask" } };
  expect(
    await execute("prepare_semantic_type_draft", args, "prepare-type")
  ).toEqual({
    kind: "tool_result",
    result: JSON.stringify({
      draftId: "semantic-draft-1",
      draft: {
        id: "new-type",
        title: "New type",
        algorithm: { kind: "md5-mask" },
      },
    }),
  });
  expect(semanticApi.prepare).toHaveBeenCalledWith(args, {
    workspaceName: "workspaces/example",
    userName: "users/member@example.com",
    chatId: "chat",
  });
});

test("blocks exemption scope choices when current operation permissions are denied", async () => {
  semanticApi.setup.mockResolvedValue({
    features: { masking: "available" },
    steps: [
      {
        id: "masking-exemptions",
        readAccess: { status: "allowed" },
        configure: {
          status: "denied",
          missingPermissions: ["bb.policies.updateMaskingExemptionPolicy"],
        },
      },
    ],
  });
  const execute = createToolExecutor({} as AppRouterInstance, {
    context: "projects/sample",
    workspaceName: "workspaces/example",
  });
  const result = await execute("ask_masking_exemption_scope", {}, "scope");
  expect(result).toEqual({
    kind: "tool_result",
    result: expect.stringContaining("bb.policies.updateMaskingExemptionPolicy"),
  });
});

beforeEach(() => {
  useAppStore.setState({
    currentUser: { name: "users/member@example.com" } as ReturnType<
      typeof useAppStore.getState
    >["currentUser"],
    workspaceResourceName: () => "workspaces/example",
  });
  const access = {
    status: "allowed",
    missingPermissions: [],
    requiredPermissions: [],
    scope: "workspaces/example",
  };
  semanticApi.setup.mockResolvedValue({
    features: { masking: "available" },
    scopeAccess: { project: access, databasePicker: access, projects: access },
    steps: ["semantic-types", "global-masking", "masking-exemptions"].map(
      (id) => ({ id, readAccess: access, configure: access })
    ),
  });
  semanticApi.setting.mockResolvedValue({
    value: { value: { case: "semanticType", value: { types: [] } } },
  });
  semanticApi.prepare.mockResolvedValue({
    draftId: "semantic-draft-1",
    draft: { id: "new-type", algorithm: { salt: "never-leak" } },
  });
  semanticApi.summary.mockReturnValue({
    id: "new-type",
    title: "New type",
    algorithm: { kind: "md5-mask" },
  });
  maskingOverviewApi.create.mockClear();
  resourceChoiceApi.validate.mockReset();
  vi.mocked(checkPreparedApiPermission).mockReset();
  vi.mocked(checkPreparedApiPermission).mockResolvedValue({ kind: "allowed" });
  projectApi.getProject.mockReset();
  vi.mocked(dispatchPreparedApiCall).mockClear();
  projectApi.getProject.mockResolvedValue({
    name: "projects/referenced",
    state: State.ACTIVE,
  });
  issueApi.getIssue.mockReset();
  issueApi.getIssue.mockResolvedValue(
    create(IssueSchema, {
      name: "projects/referenced/issues/12",
      title: "Change account table",
      type: Issue_Type.DATABASE_CHANGE,
      status: IssueStatus.OPEN,
      approvalStatus: ApprovalStatus.PENDING,
      approvalTemplate: create(ApprovalTemplateSchema, {
        flow: create(ApprovalFlowSchema, { roles: ["roles/PROJECT_OWNER"] }),
      }),
      plan: "projects/referenced/plans/12",
    })
  );
  databaseApi.getDatabase.mockReset();
  planApi.getPlan.mockReset();
  planApi.getPlan.mockResolvedValue({
    name: "projects/referenced/plans/12",
    state: State.ACTIVE,
  });
  databaseApi.getDatabase.mockResolvedValue({
    name: "instances/prod/databases/app",
    project: "projects/referenced",
    state: State.ACTIVE,
  });
  celApi.batchParse.mockReset();
});

const ident = (name: string): Expr =>
  create(ExprSchema, {
    exprKind: {
      case: "identExpr",
      value: create(Expr_IdentSchema, { name }),
    },
  });
const literal = (value: string): Expr =>
  create(ExprSchema, {
    exprKind: {
      case: "constExpr",
      value: create(ConstantSchema, {
        constantKind: { case: "stringValue", value },
      }),
    },
  });
const call = (fn: string, ...args: Expr[]): Expr =>
  create(ExprSchema, {
    exprKind: {
      case: "callExpr",
      value: create(Expr_CallSchema, { function: fn, args }),
    },
  });
const projectFilter = (project: string): Expr =>
  call("_==_", ident("project"), literal(project));

describe.each(["prepare_masking_rule_draft", "prepare_masking_rule_drafts"])(
  "%s project boundary",
  (toolName) => {
    const rule = (scope: Record<string, unknown>) => ({
      title: "Mask email",
      condition: 'resource.column_name == "email"',
      semanticType: "bb.default",
      scope,
    });
    const args = (scope: Record<string, unknown>) =>
      toolName === "prepare_masking_rule_drafts"
        ? { rules: [rule(scope), rule(scope)] }
        : rule(scope);
    const prepare =
      toolName === "prepare_masking_rule_drafts"
        ? maskingDraftApi.prepareMany
        : maskingDraftApi.prepare;

    beforeEach(() => {
      maskingDraftApi.prepare.mockClear();
      maskingDraftApi.prepareMany.mockClear();
    });

    describe.each([
      {
        context: "workspaces/test",
        referencedProject: "projects/referenced",
      },
      { context: "projects/referenced", referencedProject: undefined },
      {
        context: "projects/other",
        referencedProject: "projects/referenced",
      },
      {
        context: "projects/referenced",
        referencedProject: "projects/invalid/issues/1",
      },
    ])("bound by $context and $referencedProject", (binding) => {
      test.each(["project", "database"])(
        "rejects a foreign %s scope before preparing any draft",
        async (kind) => {
          const execute = createToolExecutor({} as AppRouterInstance, {
            chatId: "chat-1",
            userName: "users/1",
            workspaceName: "workspaces/test",
            ...binding,
          });
          const result = await execute(
            toolName,
            args({
              kind,
              project: "projects/other",
              ...(kind === "database"
                ? { database: "instances/prod/databases/app" }
                : {}),
            }),
            "foreign-project"
          );
          expect(
            result.kind === "tool_result" && JSON.parse(result.result).error
          ).toMatch(/bound project/i);
          expect(prepare).not.toHaveBeenCalled();
        }
      );

      test.each([
        { kind: "workspace" },
        { kind: "project", project: "projects/referenced" },
        {
          kind: "database",
          project: "projects/referenced",
          database: "instances/prod/databases/app",
        },
      ])("allows a confirmed $kind scope", async (scope) => {
        const execute = createToolExecutor({} as AppRouterInstance, {
          chatId: "chat-1",
          userName: "users/1",
          workspaceName: "workspaces/test",
          ...binding,
        });
        const proposal = args(scope);
        const result = await execute(toolName, proposal, "permitted-project");
        expect(
          result.kind === "tool_result" && JSON.parse(result.result)
        ).toEqual(expect.objectContaining({ draftId: expect.any(String) }));
        expect(prepare).toHaveBeenCalledWith(
          proposal,
          {
            chatId: "chat-1",
            userName: "users/1",
            workspaceName: "workspaces/test",
          },
          { requireSourceMatch: true, source: undefined }
        );
      });
    });

    test.each(["project", "database"])(
      "allows an unbound workspace chat to choose a %s scope",
      async (kind) => {
        const execute = createToolExecutor({} as AppRouterInstance, {
          chatId: "chat-1",
          userName: "users/1",
          workspaceName: "workspaces/test",
          context: "workspaces/test",
        });
        const result = await execute(
          toolName,
          args({
            kind,
            project: "projects/chosen",
            ...(kind === "database"
              ? { database: "instances/prod/databases/app" }
              : {}),
          }),
          "choose-project"
        );
        expect(
          result.kind === "tool_result" && JSON.parse(result.result)
        ).toEqual(expect.objectContaining({ draftId: expect.any(String) }));
      }
    );
  }
);

test.each([0, 1])(
  "rejects a foreign project in batch entry %i before preparing the batch",
  async (foreignIndex) => {
    maskingDraftApi.prepareMany.mockClear();
    const execute = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/1",
      workspaceName: "workspaces/test",
      referencedProject: "projects/referenced",
    });
    const result = await execute(
      "prepare_masking_rule_drafts",
      {
        rules: [0, 1].map((index) => ({
          title: "Mask email",
          condition: 'resource.column_name == "email"',
          semanticType: "bb.default",
          scope: {
            kind: "project",
            project:
              index === foreignIndex ? "projects/other" : "projects/referenced",
          },
        })),
      },
      "mixed-batch"
    );
    expect(
      result.kind === "tool_result" && JSON.parse(result.result).error
    ).toMatch(/bound project/i);
    expect(maskingDraftApi.prepareMany).not.toHaveBeenCalled();
  }
);

describe("done tool", () => {
  test("identifies current unsaved existing-rule fields as review-only", async () => {
    const snapshot = {
      sessionId: "editor-existing",
      revision: "2",
      mode: "edit" as const,
      title: "Email",
      condition: 'resource.column_name == "email"',
      semanticType: "bb.default",
    };
    maskingDraftApi.snapshot.mockResolvedValueOnce(snapshot);
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/1",
      workspaceName: "workspaces/test",
    });

    const result = await executor("get_masking_rule_draft", {}, "read-edit");
    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result)).toMatchObject({
        status: "unsaved",
        reviewOnly: true,
        draft: snapshot,
        explanation: expect.stringMatching(/existing.*review-only/i),
      });
      expect(JSON.parse(result.result)).not.toHaveProperty("draftId");
    }

    maskingDraftApi.prepare.mockRejectedValueOnce(
      new Error("Existing masking rule editors are review-only.")
    );
    const prepared = await executor(
      "prepare_masking_rule_draft",
      {},
      "prepare-edit"
    );
    expect(maskingDraftApi.prepare).toHaveBeenLastCalledWith(
      {},
      expect.anything(),
      { requireSourceMatch: true, source: snapshot }
    );
    expect(prepared.kind).toBe("tool_result");
    if (prepared.kind === "tool_result") {
      expect(JSON.parse(prepared.result)).toEqual({
        error: "Existing masking rule editors are review-only.",
      });
      expect(JSON.parse(prepared.result)).not.toHaveProperty("draftId");
    }
  });

  test("anchors a refinement to the unsaved snapshot actually read by this run", async () => {
    const snapshot = {
      sessionId: "editor-1",
      revision: "2",
      title: "Email",
      condition: 'resource.column_name == "email"',
      semanticType: "bb.default",
      chatId: "chat-1",
    };
    maskingDraftApi.snapshot.mockResolvedValueOnce(snapshot);
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/1",
      workspaceName: "workspaces/test",
    });
    await executor("get_masking_rule_draft", {}, "read");
    await executor("prepare_masking_rule_draft", {}, "prepare");
    expect(maskingDraftApi.prepare).toHaveBeenLastCalledWith(
      {},
      expect.anything(),
      { requireSourceMatch: true, source: snapshot }
    );
  });

  test.each(["prepare_masking_rule_drafts", "prepare_masking_exemption_draft"])(
    "prepares %s without writing policy in read-only chats",
    async (name) => {
      const options = {
        chatId: "chat-1",
        userName: "users/1",
        workspaceName: "workspaces/test",
        context: "projects/sample",
        runMode: "read-only" as const,
        maskingSections: ["rule", "exemption"] as const,
      };
      expect(
        getToolDefinitions(options).some((tool) => tool.name === name)
      ).toBe(true);
      const executor = createToolExecutor({} as AppRouterInstance, options);
      const args =
        name === "prepare_masking_rule_drafts"
          ? { rules: [{}, {}] }
          : {
              project: "projects/sample",
              members: ["self"],
              scope: { kind: "project" },
              condition: 'resource.column_name == "email"',
              description: "Temporary access",
              expirationTimestamp: null,
            };
      const result = await executor(name, args, "prepare");
      expect(result).toMatchObject({ kind: "tool_result" });
      expect(
        name === "prepare_masking_rule_drafts"
          ? maskingDraftApi.prepareMany
          : maskingDraftApi.prepareExemption
      ).toHaveBeenCalledWith(
        args,
        {
          chatId: options.chatId,
          userName: options.userName,
          workspaceName: options.workspaceName,
        },
        { requireSourceMatch: true, source: undefined }
      );
    }
  );

  test("rejects exemption proposals for a different bound project before preparing", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/1",
      workspaceName: "workspaces/test",
      context: "projects/sample",
    });
    const result = await executor(
      "prepare_masking_exemption_draft",
      { project: "projects/other" },
      "prepare"
    );
    expect(maskingDraftApi.prepareExemption).not.toHaveBeenCalled();
    expect(
      result.kind === "tool_result" && JSON.parse(result.result).error
    ).toMatch(/project/i);
  });

  test.each([
    "get_masking_exemption_draft",
    "prepare_masking_exemption_draft",
    "prepare_masking_rule_drafts",
  ])("does not inspect or prepare %s from a background chat", async (name) => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/1",
      workspaceName: "workspaces/test",
      canUseCurrentPage: () => false,
    });
    const result = await executor(name, {}, "background");
    expect(
      result.kind === "tool_result" && JSON.parse(result.result).error
    ).toMatch(/not selected/i);
    expect(maskingDraftApi.exemptionSnapshot).not.toHaveBeenCalled();
    expect(maskingDraftApi.prepareExemption).not.toHaveBeenCalled();
    expect(maskingDraftApi.prepareMany).not.toHaveBeenCalled();
  });

  test("prepares a local masking draft with executor-owned identity", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/1",
      workspaceName: "workspaces/test",
      context: "projects/sample",
    });
    const args = {
      title: "Mask email",
      condition: 'resource.column_name == "email"',
      semanticType: "bb.default",
      scope: { kind: "workspace" },
    };
    const result = await executor("prepare_masking_rule_draft", args, "1");
    expect(result).toMatchObject({ kind: "tool_result" });
    expect(maskingDraftApi.prepare).toHaveBeenCalledWith(
      args,
      {
        chatId: "chat-1",
        userName: "users/1",
        workspaceName: "workspaces/test",
      },
      { requireSourceMatch: true, source: undefined }
    );
  });

  test("does not inspect another chat's unsaved masking editor", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/1",
      workspaceName: "workspaces/test",
      canUseCurrentPage: () => false,
    });
    const result = await executor("get_masking_rule_draft", {}, "1");
    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result)).toHaveProperty("error");
    }
    expect(maskingDraftApi.snapshot).not.toHaveBeenCalled();
  });

  test("requires an explicit task success value", async () => {
    const definition = getToolDefinitions().find(
      (tool) => tool.name === "done"
    );
    expect(definition?.parametersSchema).toMatchObject({
      required: ["text", "success", "actions"],
    });

    const executeTool = createToolExecutor({} as AppRouterInstance);
    await expect(
      executeTool("done", { text: "Finished" }, "tool-done")
    ).rejects.toThrow(/success/i);
    await expect(
      executeTool(
        "done",
        { text: "Could not finish", success: false },
        "tool-done"
      )
    ).resolves.toEqual({
      kind: "done",
      text: "Could not finish",
      success: false,
    });
  });
});

describe("masking overview tool", () => {
  const maskingPage = { kind: "global-masking" as const };

  test("is available after loading overview from ordinary chats and masking pages", () => {
    expect(
      getToolDefinitions().some((tool) => tool.name === "get_masking_overview")
    ).toBe(false);
    expect(
      getToolDefinitions({ maskingSections: ["overview"] }).some(
        (tool) => tool.name === "get_masking_overview"
      )
    ).toBe(true);
    expect(
      getToolDefinitions({ maskingPage, maskingSections: ["overview"] }).some(
        (tool) => tool.name === "get_masking_overview"
      )
    ).toBe(true);
  });

  test("can inspect captured masking scope while another chat is selected", async () => {
    const runOverview = vi.fn(async () => "masking_overview");
    maskingOverviewApi.create.mockReturnValue(runOverview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      maskingPage,
      canUseCurrentPage: () => false,
    });
    const result = await executor("get_masking_overview", {}, "masking-1");

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(result.result).toBe("masking_overview");
    }
    expect(runOverview).toHaveBeenCalledOnce();
  });

  test("supports an optional exact semantic type lookup without widening the default catalog", async () => {
    const definition = getToolDefinitions({
      maskingSections: ["overview"],
    }).find((tool) => tool.name === "get_masking_overview");
    expect(definition?.parametersSchema).toMatchObject({
      properties: {
        semanticTypeId: { type: "string", minLength: 1, maxLength: 500 },
      },
    });
    expect(definition?.parametersSchema.required ?? []).not.toContain(
      "semanticTypeId"
    );
    const runOverview = vi.fn(async () => "exact_semantic_type");
    maskingOverviewApi.create.mockReturnValue(runOverview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      workspaceName: "workspaces/default",
    });
    await expect(
      executor(
        "get_masking_overview",
        { semanticTypeId: "custom-149" },
        "masking-exact"
      )
    ).resolves.toEqual({ kind: "tool_result", result: "exact_semantic_type" });
    expect(runOverview).toHaveBeenCalledWith({ semanticTypeId: "custom-149" });
  });

  test("runs the bounded snapshot for the selected masking page", async () => {
    const runOverview = vi.fn(async () => "masking_overview");
    maskingOverviewApi.create.mockReturnValue(runOverview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      maskingPage,
      workspaceName: "workspaces/default",
      canUseCurrentPage: () => true,
    });

    await expect(
      executor("get_masking_overview", {}, "masking-2")
    ).resolves.toEqual({ kind: "tool_result", result: "masking_overview" });
    expect(maskingOverviewApi.create).toHaveBeenCalledWith({
      page: maskingPage,
      workspaceName: "workspaces/default",
      context: undefined,
      databaseName: undefined,
    });
    expect(runOverview).toHaveBeenCalledOnce();
  });

  test("binds masking guidance to the selected project rather than the live page", async () => {
    const runOverview = vi.fn(async () => "masking_overview");
    maskingOverviewApi.create.mockReturnValue(runOverview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "projects/original",
      referencedProject: "projects/referenced",
      workspaceName: "workspaces/default",
      maskingPage: { kind: "masking-exemption", project: "projects/original" },
    });

    await executor(
      "get_masking_overview",
      { project: "projects/referenced" },
      "masking-3"
    );
    expect(maskingOverviewApi.create).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "projects/referenced",
      })
    );
    expect(runOverview).toHaveBeenCalledWith({
      project: "projects/referenced",
    });
  });
});

describe("agent tools referenced project", () => {
  test("scopes bounded project reads to the reference for one executor only", async () => {
    const referencedExecutor = createToolExecutor({} as AppRouterInstance, {
      context: "projects/original",
      referencedProject: "projects/referenced",
      runMode: "read-only",
    });

    await referencedExecutor("get_project_overview", {}, "tool-project");
    await referencedExecutor(
      "get_project_governance_overview",
      {},
      "tool-policy"
    );
    await referencedExecutor("get_project_open_issues", {}, "tool-issues");
    await referencedExecutor("get_project_recent_activity", {}, "tool-recent");
    await referencedExecutor(
      "get_issue_overview",
      { issueName: "projects/referenced/issues/123" },
      "tool-issue"
    );
    await referencedExecutor(
      "get_plan_overview",
      { planName: "projects/referenced/plans/123" },
      "tool-plan"
    );
    await referencedExecutor(
      "get_database_overview",
      { databaseName: "projects/referenced/instances/prod/databases/app" },
      "tool-database"
    );
    await referencedExecutor("get_database_count", {}, "tool-count");

    expect(projectApi.getProject).toHaveBeenCalledTimes(1);
    expect(projectApi.getProject).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/referenced" }),
      expect.anything()
    );
    for (const factory of [
      createProjectOverviewTool,
      createProjectGovernanceOverviewTool,
      createProjectOpenIssuesTool,
      createProjectRecentActivityTool,
      createIssueOverviewTool,
      createPlanOverviewTool,
      createDatabaseOverviewTool,
    ]) {
      expect(factory).toHaveBeenCalledWith({
        projectName: "projects/referenced",
      });
    }
    expect(createDatabaseCountTool).toHaveBeenCalledWith({
      scopeName: "projects/referenced",
    });

    createToolExecutor({} as AppRouterInstance, {
      context: "projects/original",
    });
    expect(createProjectOverviewTool).toHaveBeenLastCalledWith({
      projectName: "projects/original",
    });
  });

  test("does not read stale or inaccessible referenced projects", async () => {
    projectApi.getProject.mockRejectedValue(new Error("not found"));
    const overview = vi.fn(async () => "must not run");
    vi.mocked(createProjectOverviewTool).mockReturnValueOnce(overview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "projects/original",
      referencedProject: "projects/referenced",
    });

    expect(await executor("get_project_overview", {}, "tool-project")).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(
      await executor(
        "call_api",
        {
          operationId: "DatabaseService/ListDatabases",
          body: { parent: "projects/referenced" },
        },
        "tool-api"
      )
    ).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(overview).not.toHaveBeenCalled();
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not count databases in an inaccessible referenced project", async () => {
    projectApi.getProject.mockRejectedValue(new Error("not found"));
    const count = vi.fn(async () => "must not run");
    vi.mocked(createDatabaseCountTool).mockReturnValueOnce(count);
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "workspaces/demo",
      referencedProject: "projects/referenced",
    });

    expect(await executor("get_database_count", {}, "tool-count")).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(count).not.toHaveBeenCalled();
  });

  test("allows a selected Issue read when Issue access exists without Project access", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const overview = vi.fn(async () => "issue_overview");
    vi.mocked(createIssueOverviewTool).mockReturnValueOnce(overview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "workspaces/demo",
      referencedProject: "projects/referenced",
      referencedIssue: "projects/referenced/issues/12",
    });

    expect(
      await executor(
        "get_issue_overview",
        { issueName: "projects/referenced/issues/12" },
        "tool-issue"
      )
    ).toEqual({ kind: "tool_result", result: "issue_overview" });
    expect(issueApi.getIssue).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/referenced/issues/12" }),
      expect.anything()
    );
    expect(overview).toHaveBeenCalledWith({
      issueName: "projects/referenced/issues/12",
    });
  });

  test("allows an exact selected Plan read without Project access", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const overview = vi.fn(async () => "plan_overview");
    vi.mocked(createPlanOverviewTool).mockReturnValueOnce(overview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "workspaces/demo",
      referencedProject: "projects/referenced",
      referencedPlan: "projects/referenced/plans/12",
    });

    expect(
      await executor(
        "get_plan_overview",
        { planName: "projects/referenced/plans/12" },
        "tool-plan"
      )
    ).toEqual({ kind: "tool_result", result: "plan_overview" });
    expect(overview).toHaveBeenCalledWith({
      planName: "projects/referenced/plans/12",
    });
    expect(planApi.getPlan).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/referenced/plans/12" }),
      expect.anything()
    );

    expect(
      await executor(
        "call_api",
        {
          operationId: "PlanService/GetPlan",
          body: { name: "projects/referenced/plans/12" },
        },
        "tool-api"
      )
    ).toEqual({ kind: "tool_result", result: "call_api" });
    expect(dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("selected Plan access cannot read other Plans or use project-wide APIs", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const overview = vi.fn(async () => "must not run");
    vi.mocked(createPlanOverviewTool).mockReturnValueOnce(overview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedPlan: "projects/referenced/plans/12",
    });

    for (const [name, args] of [
      ["get_plan_overview", { planName: "projects/referenced/plans/other" }],
      ["get_project_overview", {}],
      [
        "call_api",
        {
          operationId: "PlanService/GetPlan",
          body: { name: "projects/referenced/plans/other" },
        },
      ],
      [
        "call_api",
        {
          operationId: "PlanService/UpdatePlan",
          body: { plan: { name: "projects/referenced/plans/12" } },
        },
      ],
      [
        "call_api",
        {
          operationId: "PlanService/ListPlans",
          body: { parent: "projects/referenced" },
        },
      ],
    ] as const) {
      expect(await executor(name, args, `tool-${name}`)).toEqual({
        kind: "tool_result",
        result: JSON.stringify({
          error:
            "The referenced project is unavailable or you do not have access.",
        }),
      });
    }
    expect(overview).not.toHaveBeenCalled();
    expect(planApi.getPlan).not.toHaveBeenCalled();
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not accept a deleted or mismatched Plan as access proof", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const overview = vi.fn(async () => "must not run");
    vi.mocked(createPlanOverviewTool).mockReturnValueOnce(overview);
    for (const plan of [
      { name: "projects/referenced/plans/12", state: State.DELETED },
      { name: "projects/referenced/plans/other", state: State.ACTIVE },
    ]) {
      planApi.getPlan.mockResolvedValueOnce(plan);
      const executor = createToolExecutor({} as AppRouterInstance, {
        referencedProject: "projects/referenced",
        referencedPlan: "projects/referenced/plans/12",
      });
      expect(
        await executor(
          "get_plan_overview",
          { planName: "projects/referenced/plans/12" },
          "tool-plan"
        )
      ).toEqual({
        kind: "tool_result",
        result: JSON.stringify({
          error:
            "The referenced project is unavailable or you do not have access.",
        }),
      });
    }
    expect(overview).not.toHaveBeenCalled();
  });

  test("allows a selected Issue decision to request permission without Project access", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-issue",
      userName: "users/me@example.com",
      context: "workspaces/demo",
      referencedProject: "projects/referenced",
      referencedIssue: "projects/referenced/issues/12",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/ApproveIssue",
        body: { name: "projects/referenced/issues/12" },
      },
      "tool-approve"
    );

    expect(result).toMatchObject({
      kind: "ask_user",
      ask: {
        apiPermission: {
          operationId: "bytebase.v1.IssueService.ApproveIssue",
        },
      },
    });
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not use Issue access for other project or database tools", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const projectOverview = vi.fn(async () => "must not run");
    const databaseOverview = vi.fn(async () => "must not run");
    vi.mocked(createProjectOverviewTool).mockReturnValueOnce(projectOverview);
    vi.mocked(createDatabaseOverviewTool).mockReturnValueOnce(databaseOverview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedIssue: "projects/referenced/issues/12",
    });

    for (const [name, args] of [
      ["get_project_overview", {}],
      [
        "get_database_overview",
        { databaseName: "instances/prod/databases/app" },
      ],
    ] as const) {
      const result = await executor(name, args, `tool-${name}`);
      expect(result).toEqual({
        kind: "tool_result",
        result: JSON.stringify({
          error:
            "The referenced project is unavailable or you do not have access.",
        }),
      });
    }
    expect(projectOverview).not.toHaveBeenCalled();
    expect(databaseOverview).not.toHaveBeenCalled();
    expect(issueApi.getIssue).not.toHaveBeenCalled();
  });

  test("does not use Issue access for a database-reference turn", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    databaseApi.getDatabase.mockRejectedValue(new Error("permission denied"));
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedDatabase: "instances/prod/databases/app",
      referencedIssue: "projects/referenced/issues/12",
    });

    const result = await executor(
      "get_database_count",
      { scope: "referenced-instance" },
      "tool-count"
    );

    expect(result).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(issueApi.getIssue).not.toHaveBeenCalled();
  });

  test("uses selected Database access without requiring Project get access", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const databaseOverview = vi.fn(async () => "database_overview");
    const databaseCount = vi.fn(async () => "database_count");
    vi.mocked(createDatabaseOverviewTool).mockReturnValueOnce(databaseOverview);
    vi.mocked(createDatabaseCountTool).mockReturnValueOnce(databaseCount);
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedDatabase: "instances/prod/databases/app",
    });

    expect(
      await executor(
        "get_database_overview",
        { databaseName: "instances/prod/databases/app" },
        "tool-overview"
      )
    ).toEqual({ kind: "tool_result", result: "database_overview" });
    expect(
      await executor(
        "get_database_count",
        { scope: "referenced-instance" },
        "tool-count"
      )
    ).toEqual({ kind: "tool_result", result: "database_count" });
    expect(
      await executor(
        "call_api",
        {
          operationId: "DatabaseService/ListDatabases",
          body: { parent: "projects/referenced" },
        },
        "tool-api"
      )
    ).toEqual({ kind: "tool_result", result: "call_api" });
    expect(databaseApi.getDatabase).toHaveBeenCalledTimes(1);
    expect(databaseOverview).toHaveBeenCalledWith({
      databaseName: "instances/prod/databases/app",
    });
    expect(databaseCount).toHaveBeenCalledWith({
      scope: "referenced-instance",
    });
    expect(projectApi.getProject).not.toHaveBeenCalled();
    expect(dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("does not use selected Database access for another project tool", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const projectOverview = vi.fn(async () => "must not run");
    vi.mocked(createProjectOverviewTool).mockReturnValueOnce(projectOverview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedDatabase: "instances/prod/databases/app",
    });

    expect(await executor("get_project_overview", {}, "tool-project")).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(projectOverview).not.toHaveBeenCalled();
    expect(databaseApi.getDatabase).not.toHaveBeenCalled();
  });

  test("does not accept a Database from another project as access proof", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    databaseApi.getDatabase.mockResolvedValue({
      name: "instances/prod/databases/app",
      project: "projects/other",
      state: State.ACTIVE,
    });
    const databaseOverview = vi.fn(async () => "must not run");
    vi.mocked(createDatabaseOverviewTool).mockReturnValueOnce(databaseOverview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedDatabase: "instances/prod/databases/app",
    });

    expect(
      await executor(
        "get_database_overview",
        { databaseName: "instances/prod/databases/app" },
        "tool-overview"
      )
    ).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(databaseOverview).not.toHaveBeenCalled();
  });

  test("does not use selected Database access for another Database", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const databaseOverview = vi.fn(async () => "must not run");
    vi.mocked(createDatabaseOverviewTool).mockReturnValueOnce(databaseOverview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedDatabase: "instances/prod/databases/app",
    });

    expect(
      await executor(
        "get_database_overview",
        { databaseName: "instances/prod/databases/other" },
        "tool-overview"
      )
    ).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(databaseOverview).not.toHaveBeenCalled();
    expect(databaseApi.getDatabase).not.toHaveBeenCalled();
  });

  test("does not authorize another project through a mismatched Issue reference", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedIssue: "projects/other/issues/12",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/GetIssue",
        body: { name: "projects/other/issues/12" },
      },
      "tool-other-issue"
    );

    expect(result).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(issueApi.getIssue).not.toHaveBeenCalled();
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not call an API for another project even when the selected Project is accessible", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/GetIssue",
        body: { name: "projects/other/issues/34" },
      },
      "tool-other-issue"
    );

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result).error).toMatch(/referenced project/i);
    }
    expect(projectApi.getProject).toHaveBeenCalledTimes(1);
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not let selected Database access bypass the API target scope", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedDatabase: "instances/prod/databases/app",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/GetIssue",
        body: { name: "projects/other/issues/34" },
      },
      "tool-other-issue"
    );

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result).error).toMatch(/referenced project/i);
    }
    expect(databaseApi.getDatabase).toHaveBeenCalledTimes(1);
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("scopes batch Issue status updates by Issue names, not an ignored parent", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-issue",
      userName: "users/me@example.com",
      referencedProject: "projects/referenced",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/BatchUpdateIssuesStatus",
        body: {
          parent: "projects/other",
          issues: ["projects/referenced/issues/12"],
          status: "OPEN",
        },
      },
      "tool-update-status"
    );

    expect(result).toMatchObject({
      kind: "ask_user",
      ask: {
        apiPermission: {
          operationId: "bytebase.v1.IssueService.BatchUpdateIssuesStatus",
        },
      },
    });
  });

  test("rejects a batch Issue status update targeting another Project", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-issue",
      userName: "users/me@example.com",
      referencedProject: "projects/referenced",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/BatchUpdateIssuesStatus",
        body: {
          parent: "projects/referenced",
          issues: ["projects/other/issues/34"],
          status: "OPEN",
        },
      },
      "tool-other-status"
    );

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result).error).toMatch(/referenced project/i);
    }
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not run a cross-project database search for a selected Project", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "DatabaseService/SearchDatabases",
        body: { pageSize: 25 },
      },
      "tool-search-databases"
    );

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result).error).toMatch(/referenced project/i);
    }
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("allows a database search constrained to the selected Project", async () => {
    celApi.batchParse.mockResolvedValue({
      expressions: [
        call(
          "_&&_",
          projectFilter("projects/referenced"),
          call("_==_", ident("engine"), literal("MYSQL"))
        ),
      ],
    });
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
    });

    expect(
      await executor(
        "call_api",
        {
          operationId: "DatabaseService/SearchDatabases",
          body: {
            filter: 'project == "projects/referenced" && engine == "MYSQL"',
          },
        },
        "tool-project-search"
      )
    ).toEqual({ kind: "tool_result", result: "call_api" });
    expect(celApi.batchParse).toHaveBeenCalledTimes(1);
    expect(dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
  });

  test("does not accept a database search whose filter can include another Project", async () => {
    celApi.batchParse.mockResolvedValue({
      expressions: [
        call(
          "_||_",
          projectFilter("projects/referenced"),
          projectFilter("projects/other")
        ),
      ],
    });
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "DatabaseService/SearchDatabases",
        body: {
          filter:
            'project == "projects/referenced" || project == "projects/other"',
        },
      },
      "tool-other-search"
    );

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result).error).toMatch(/referenced project/i);
    }
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not run a database search when its project filter cannot be verified", async () => {
    celApi.batchParse.mockRejectedValue(new Error("parser unavailable"));
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "DatabaseService/SearchDatabases",
        body: { filter: 'project == "projects/referenced"' },
      },
      "tool-unverified-search"
    );

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result).error).toMatch(/referenced project/i);
    }
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not call a database API for a database in another Project", async () => {
    databaseApi.getDatabase.mockResolvedValue({
      name: "instances/prod/databases/app",
      project: "projects/other",
      state: State.ACTIVE,
    });
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "SQLService/Query",
        body: {
          name: "instances/prod/databases/app",
          statement: "SELECT 1",
        },
      },
      "tool-other-database"
    );

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result).error).toMatch(/referenced project/i);
    }
    expect(databaseApi.getDatabase).toHaveBeenCalledTimes(1);
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not use selected Issue access for an unrelated API target", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedIssue: "projects/referenced/issues/12",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/GetIssue",
        body: { name: "projects/other/issues/34" },
      },
      "tool-other-issue"
    );

    expect(result).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("does not accept a different Issue from GetIssue as access proof", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    issueApi.getIssue.mockResolvedValue({
      name: "projects/referenced/issues/13",
    });
    const executor = createToolExecutor({} as AppRouterInstance, {
      referencedProject: "projects/referenced",
      referencedIssue: "projects/referenced/issues/12",
    });

    const result = await executor(
      "get_issue_overview",
      { issueName: "projects/referenced/issues/12" },
      "tool-issue"
    );

    expect(result).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        error:
          "The referenced project is unavailable or you do not have access.",
      }),
    });
  });

  test("allows a comment on the selected Issue without Project read access", async () => {
    projectApi.getProject.mockRejectedValue(new Error("permission denied"));
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-issue",
      userName: "users/me@example.com",
      referencedProject: "projects/referenced",
      referencedIssue: "projects/referenced/issues/12",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/CreateIssueComment",
        body: {
          parent: "projects/referenced/issues/12",
          issueComment: { comment: "Looks ready." },
        },
      },
      "tool-comment"
    );

    expect(result).toMatchObject({
      kind: "ask_user",
      ask: {
        apiPermission: {
          operationId: "bytebase.v1.IssueService.CreateIssueComment",
        },
      },
    });
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("passes only the current database reference to the instance count tool", async () => {
    const count = vi.fn(async () => "instance_count");
    vi.mocked(createDatabaseCountTool).mockReturnValueOnce(count);
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "projects/original",
      referencedProject: "projects/referenced",
      referencedDatabase: "instances/shared/databases/selected",
      runMode: "read-only",
    });

    await expect(
      executor(
        "get_database_count",
        { scope: "referenced-instance" },
        "tool-instance-count"
      )
    ).resolves.toEqual({ kind: "tool_result", result: "instance_count" });
    expect(createDatabaseCountTool).toHaveBeenCalledWith({
      scopeName: "projects/referenced",
      referencedProject: "projects/referenced",
      referencedDatabase: "instances/shared/databases/selected",
    });
    expect(count).toHaveBeenCalledWith({ scope: "referenced-instance" });
    expect(projectApi.getProject).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/referenced" }),
      expect.anything()
    );
  });

  test("does not read a soft-deleted referenced project", async () => {
    projectApi.getProject.mockResolvedValue({
      name: "projects/referenced",
      state: State.DELETED,
    });
    const overview = vi.fn(async () => "must not run");
    vi.mocked(createProjectOverviewTool).mockReturnValueOnce(overview);
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "projects/original",
      referencedProject: "projects/referenced",
    });

    const result = await executor("get_project_overview", {}, "tool-project");

    expect(result).toMatchObject({ kind: "tool_result" });
    expect(
      JSON.parse(result.kind === "tool_result" ? result.result : "{}")
    ).toHaveProperty("error");
    expect(overview).not.toHaveBeenCalled();
  });

  test("reference turns can discover and call APIs without page actions", async () => {
    const definitions = getToolDefinitions({
      referencedProject: "projects/referenced",
    });
    expect(definitions.map((tool) => tool.name)).toEqual(
      expect.arrayContaining(["search_api", "call_api"])
    );
    expect(definitions.map((tool) => tool.name)).not.toEqual(
      expect.arrayContaining(["navigate", "dom_action"])
    );
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "projects/original",
      referencedProject: "projects/referenced",
    });

    expect(
      await executor("search_api", { query: "list databases" }, "tool-search")
    ).toEqual({
      kind: "tool_result",
      result: "search_api",
    });
    expect(
      await executor(
        "call_api",
        {
          operationId: "DatabaseService/ListDatabases",
          body: { parent: "projects/referenced", pageSize: 25 },
        },
        "tool-read"
      )
    ).toEqual({
      kind: "tool_result",
      result: "call_api",
    });
    expect(dispatchPreparedApiCall).toHaveBeenCalledTimes(1);
    expect(dispatchPreparedApiCall).toHaveBeenCalledWith(
      expect.objectContaining({
        operationId: "bytebase.v1.DatabaseService.ListDatabases",
        body: JSON.stringify({ parent: "projects/referenced", pageSize: 25 }),
      })
    );

    for (const name of ["navigate", "dom_action"]) {
      const result = await executor(name, {}, `tool-${name}`);
      expect(result.kind).toBe("tool_result");
      if (result.kind === "tool_result") {
        expect(JSON.parse(result.result)).toHaveProperty("error");
      }
    }
  });

  test("asks before a referenced-project write API request", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/me@example.com",
      context: "projects/original",
      referencedProject: "projects/referenced",
      runMode: "read-only",
    });
    const args = {
      operationId: "ProjectService/UpdateProject",
      body: {
        project: {
          name: "projects/referenced",
          title: "Renamed project",
        },
        updateMask: { paths: ["title"] },
      },
    };

    expect(await executor("call_api", args, "tool-api")).toMatchObject({
      kind: "ask_user",
      ask: {
        kind: "confirm",
        toolCallId: "tool-api",
        apiPermission: {
          operationId: "bytebase.v1.ProjectService.UpdateProject",
        },
      },
    });
    expect(projectApi.getProject).toHaveBeenCalledTimes(1);
    expect(projectApi.getProject).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/referenced" }),
      expect.anything()
    );
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("refuses an IAM-denied write before requesting user confirmation", async () => {
    vi.mocked(checkPreparedApiPermission).mockResolvedValue({
      kind: "denied",
      result: JSON.stringify({
        status: 403,
        error:
          "Not called: bb.projects.update is not granted on projects/referenced.",
      }),
    });
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/me@example.com",
      context: "projects/referenced",
    });
    const result = await executor(
      "call_api",
      {
        operationId: "ProjectService/UpdateProject",
        body: {
          project: { name: "projects/referenced", title: "Changed" },
          updateMask: "title",
        },
      },
      "tool-denied"
    );
    expect(result).toMatchObject({
      kind: "tool_result",
      result: expect.stringContaining("Not called: bb.projects.update"),
    });
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("refuses an IAM-denied read without calling the target API", async () => {
    vi.mocked(checkPreparedApiPermission).mockResolvedValue({
      kind: "denied",
      result: JSON.stringify({
        status: 403,
        error: "Not called: bb.projects.get is not granted.",
      }),
    });
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "projects/referenced",
    });
    const result = await executor(
      "call_api",
      {
        operationId: "ProjectService/GetProject",
        body: { name: "projects/referenced" },
      },
      "tool-denied-read"
    );
    expect(result).toMatchObject({
      kind: "tool_result",
      result: expect.stringContaining("Not called: bb.projects.get"),
    });
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("previews the selected Issue approval and potential rollout in permission", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-issue",
      userName: "users/me@example.com",
      context: "projects/original",
      referencedProject: "projects/referenced",
      referencedIssue: "projects/referenced/issues/12",
      runMode: "read-only",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/ApproveIssue",
        body: { name: "projects/referenced/issues/12" },
      },
      "tool-approve"
    );

    expect(result).toMatchObject({
      kind: "ask_user",
      ask: {
        apiPermission: {
          operationId: "bytebase.v1.IssueService.ApproveIssue",
        },
      },
    });
    if (result.kind !== "ask_user" || !result.ask.apiPermission) {
      throw new Error("Expected permission request");
    }
    expect(
      getPendingApiCallPreview({
        actionId: result.ask.apiPermission.actionId,
        chatId: "chat-issue",
        toolCallId: "tool-approve",
        userName: "users/me@example.com",
      })
    ).toMatchObject({
      issueDecision: {
        issueName: "projects/referenced/issues/12",
        issueTitle: "Change account table",
        impact: "final-approval-with-rollout",
      },
    });
    expect(issueApi.getIssue).toHaveBeenCalledTimes(1);
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("blocks an Issue decision that targets a different Issue reference", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-issue",
      userName: "users/me@example.com",
      referencedProject: "projects/referenced",
      referencedIssue: "projects/referenced/issues/12",
    });

    const result = await executor(
      "call_api",
      {
        operationId: "IssueService/RejectIssue",
        body: { name: "projects/referenced/issues/13" },
      },
      "tool-reject"
    );

    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result).error).toMatch(/selected Issue/);
    }
    expect(issueApi.getIssue).not.toHaveBeenCalled();
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("asks before SQLService/Query even though its RPC is classified READ", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      chatId: "chat-1",
      userName: "users/me@example.com",
    });

    expect(
      await executor(
        "call_api",
        {
          operationId: "SQLService/Query",
          body: {
            name: "instances/example/databases/app",
            statement: "UPDATE accounts SET active = false",
          },
        },
        "tool-query"
      )
    ).toMatchObject({
      kind: "ask_user",
      ask: {
        kind: "confirm",
        toolCallId: "tool-query",
        apiPermission: { operationId: "bytebase.v1.SQLService.Query" },
      },
    });
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("previews and dispatches the same default-limited query after approval", async () => {
    const identity = {
      chatId: "chat-query-preview",
      userName: "users/me@example.com",
      toolCallId: "tool-query-preview",
    };
    const executor = createToolExecutor({} as AppRouterInstance, identity);
    const body: Record<string, unknown> = {
      name: "instances/example/databases/app",
      statement: "SELECT * FROM accounts",
    };

    const result = await executor(
      "call_api",
      { operationId: "SQLService/Query", body },
      identity.toolCallId
    );

    expect(result.kind).toBe("ask_user");
    if (result.kind !== "ask_user" || !result.ask.apiPermission) {
      throw new Error("Expected API permission request");
    }
    const actionId = result.ask.apiPermission.actionId;
    const preview = getPendingApiCallPreview({ ...identity, actionId });
    expect(preview).toMatchObject({
      operationId: "bytebase.v1.SQLService.Query",
      body: JSON.stringify({ ...body, limit: 100 }),
    });
    expect(body).not.toHaveProperty("limit");
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();

    body.limit = 500;
    body.statement = "SELECT * FROM other_accounts";
    await resolvePendingApiCall({ ...identity, actionId, approved: true });

    expect(dispatchPreparedApiCall).toHaveBeenCalledExactlyOnceWith({
      operationId: "bytebase.v1.SQLService.Query",
      path: "/bytebase.v1.SQLService/Query",
      body: preview?.body,
    });
  });

  test("refuses a write API when no signed-in chat can approve it", async () => {
    const executor = createToolExecutor({} as AppRouterInstance);

    expect(
      await executor(
        "call_api",
        {
          operationId: "ProjectService/UpdateProject",
          body: { project: { name: "projects/example", title: "Changed" } },
        },
        "tool-write"
      )
    ).toEqual({
      kind: "tool_result",
      result: expect.stringContaining("Cannot request API permission"),
    });
    expect(dispatchPreparedApiCall).not.toHaveBeenCalled();
  });

  test("rejects malformed references and keeps the chat-bound project", async () => {
    const executor = createToolExecutor({} as AppRouterInstance, {
      context: "projects/original",
      referencedProject: "projects/referenced/issues/123",
    });
    await executor("get_project_overview", {}, "tool-project");

    expect(projectApi.getProject).not.toHaveBeenCalled();
    expect(createProjectOverviewTool).toHaveBeenLastCalledWith({
      projectName: "projects/original",
    });
  });

  test("describes the one-turn reference without reclassifying the chat", () => {
    const definitions = getToolDefinitions({
      referencedProject: "projects/referenced",
      runMode: "read-only",
    });
    const projectOverview = definitions.find(
      (tool) => tool.name === "get_project_overview"
    );
    const workspaceOverview = definitions.find(
      (tool) => tool.name === "get_workspace_overview"
    );

    expect(projectOverview?.description).toContain("projects/referenced");
    expect(projectOverview?.description).toContain("this turn");
    expect(workspaceOverview?.description).toContain(
      "workspace bound to this chat"
    );
    expect(
      getToolDefinitions({ referencedProject: "projects/referenced" }).map(
        (tool) => tool.name
      )
    ).toEqual(
      expect.arrayContaining([
        "get_project_open_issues",
        "get_project_recent_activity",
      ])
    );
  });
});

describe("agent tools navigate", () => {
  test("calls onNavigate after a successful navigation result", async () => {
    const navigateTool = vi.fn(async () =>
      JSON.stringify({
        navigated: true,
        currentPath: "/projects/demo",
      })
    );
    vi.mocked(createNavigateTool).mockReturnValue(navigateTool);
    const onNavigate = vi.fn();
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      onNavigate,
    });

    const result = await executeTool(
      "navigate",
      {
        path: "/projects/demo",
      },
      "tool-navigate"
    );

    expect(result).toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        navigated: true,
        currentPath: "/projects/demo",
      }),
    });
    expect(onNavigate).toHaveBeenCalledTimes(1);
  });
});

describe("agent tools get_page_state", () => {
  test("describes a ref-labeled DOM tree", () => {
    const pageState = getToolDefinitions().find(
      (tool) => tool.name === "get_page_state"
    );

    expect(pageState).toBeDefined();
    expect(pageState?.description).toContain("ref-labeled DOM tree");
    expect(pageState?.description).toContain("element refs like [e1]");
    expect(pageState?.description).not.toContain("element indices");
    expect(pageState?.parametersSchema).toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          mode: expect.objectContaining({
            description: expect.stringContaining("ref-labeled tree"),
          }),
        }),
      })
    );
  });
});

describe("agent tools dom_action", () => {
  test("exposes ref-based dom_action schema", () => {
    const domAction = getToolDefinitions().find(
      (tool) => tool.name === "dom_action"
    );

    expect(domAction).toBeDefined();
    expect(domAction?.description).toContain(
      '**Always call get_page_state(mode="dom") first** to get the element ref, such as [e1].'
    );
    expect(domAction?.parametersSchema).toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          type: expect.objectContaining({
            enum: ["click", "input", "select", "read", "scroll"],
          }),
          ref: expect.objectContaining({
            type: "string",
          }),
        }),
        required: ["type", "ref"],
      })
    );
    expect(domAction?.parametersSchema).not.toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          index: expect.anything(),
        }),
      })
    );
  });
});

describe("agent tools read-only mode", () => {
  test("describes database counts as bounded and partial", () => {
    const tool = getToolDefinitions({ runMode: "read-only" }).find(
      ({ name }) => name === "get_database_count"
    );
    expect(tool?.description).toContain("reads one page");
    expect(tool?.description).toContain('report "at least N"');
    expect(tool?.description).not.toContain("follows every");
  });

  test("exposes instance sync overview only for an instance detail page", async () => {
    expect(
      getToolDefinitions({ runMode: "read-only" }).map((tool) => tool.name)
    ).not.toContain("get_instance_overview");
    expect(
      getToolDefinitions({
        runMode: "read-only",
        instanceName: "instances/primary",
      }).map((tool) => tool.name)
    ).toContain("get_instance_overview");

    const executor = createToolExecutor({} as AppRouterInstance, {
      runMode: "read-only",
      pagePath: "/instances/primary",
    });
    await expect(
      executor("get_instance_overview", {}, "tool-instance-overview")
    ).resolves.toEqual({ kind: "tool_result", result: "instance_overview" });

    const unboundExecutor = createToolExecutor({} as AppRouterInstance, {
      runMode: "read-only",
    });
    const result = await unboundExecutor(
      "get_instance_overview",
      {},
      "tool-unbound-instance-overview"
    );
    expect(result.kind).toBe("tool_result");
    if (result.kind === "tool_result") {
      expect(JSON.parse(result.result)).toHaveProperty("error");
    }
  });

  test("only exposes page inspection, guidance, and completion", () => {
    expect(
      getToolDefinitions({ runMode: "read-only" }).map((tool) => tool.name)
    ).toEqual([
      "get_page_state",
      "get_project_overview",
      "get_project_governance_overview",
      "get_issue_overview",
      "get_plan_overview",
      "get_database_overview",
      "get_database_count",
      "get_workspace_overview",
      "get_project_open_issues",
      "get_project_recent_activity",
      "get_workspace_attention",
      "get_workspace_recent_activity",
      "get_skill",
      "ask_user",
      "done",
    ]);
    expect(getToolDefinitions().map((tool) => tool.name)).not.toContain(
      "get_project_open_issues"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).not.toContain(
      "get_project_recent_activity"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).not.toContain(
      "get_workspace_recent_activity"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).not.toContain(
      "get_workspace_attention"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).toContain(
      "get_project_overview"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).toContain(
      "get_project_governance_overview"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).toContain(
      "get_issue_overview"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).toContain(
      "get_plan_overview"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).toContain(
      "get_database_overview"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).toContain(
      "get_database_count"
    );
  });

  test("keeps loaded local masking workflows available without page or API writes", () => {
    const names = getToolDefinitions({
      runMode: "read-only",
      maskingSections: ["rule", "exemption", "semantic-type"],
    }).map((tool) => tool.name);

    expect(names).toEqual(
      expect.arrayContaining([
        "get_masking_overview",
        "get_masking_rule_draft",
        "prepare_masking_rule_draft",
        "prepare_masking_rule_drafts",
        "ask_masking_rule_scope",
        "get_masking_exemption_draft",
        "prepare_masking_exemption_draft",
        "ask_masking_exemption_scope",
        "ask_masking_semantic_type",
        "prepare_semantic_type_draft",
      ])
    );
    expect(names).not.toContain("call_api");
    expect(names).not.toContain("dom_action");
    expect(names).not.toContain("navigate");
    expect(new Set(names).size).toBe(names.length);
  });

  test.each([
    ["workspace", "workspaces/default"],
    ["project", "projects/sample"],
  ])("uses the chat-bound %s for a database count", async (_, context) => {
    const count = vi.fn(async () => "database_count");
    vi.mocked(createDatabaseCountTool).mockReturnValueOnce(count);
    const executor = createToolExecutor({} as AppRouterInstance, {
      context,
      runMode: "read-only",
    });

    await expect(
      executor("get_database_count", {}, "tool-count")
    ).resolves.toEqual({
      kind: "tool_result",
      result: "database_count",
    });
    expect(createDatabaseCountTool).toHaveBeenLastCalledWith({
      scopeName: context,
    });
    expect(count).toHaveBeenCalledOnce();
  });

  test("uses the chat-bound project for a generic overview lookup", async () => {
    const overviewTool = vi.fn(async () => "project_overview");
    vi.mocked(createProjectOverviewTool).mockReturnValue(overviewTool);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      runMode: "read-only",
    });

    await expect(
      executeTool("get_project_overview", {}, "tool-project-overview")
    ).resolves.toEqual({
      kind: "tool_result",
      result: "project_overview",
    });
    expect(createProjectOverviewTool).toHaveBeenCalledWith({
      projectName: "projects/sample",
    });
    expect(overviewTool).toHaveBeenCalledWith();
  });

  test("uses the chat-bound workspace for a generic overview lookup", async () => {
    const overviewTool = vi.fn(async () => "workspace_overview");
    vi.mocked(createWorkspaceOverviewTool).mockReturnValue(overviewTool);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "workspaces/default",
      runMode: "read-only",
    });

    await expect(
      executeTool("get_workspace_overview", {}, "tool-workspace-overview")
    ).resolves.toEqual({
      kind: "tool_result",
      result: "workspace_overview",
    });
    expect(createWorkspaceOverviewTool).toHaveBeenCalledWith({
      workspaceName: "workspaces/default",
    });
    expect(overviewTool).toHaveBeenCalledWith();
  });

  test("uses the chat-bound project for a governance overview lookup", async () => {
    const governanceTool = vi.fn(async () => "project_governance_overview");
    vi.mocked(createProjectGovernanceOverviewTool).mockReturnValue(
      governanceTool
    );
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      runMode: "read-only",
    });

    await expect(
      executeTool(
        "get_project_governance_overview",
        {},
        "tool-project-governance-overview"
      )
    ).resolves.toEqual({
      kind: "tool_result",
      result: "project_governance_overview",
    });
    expect(createProjectGovernanceOverviewTool).toHaveBeenCalledWith({
      projectName: "projects/sample",
    });
    expect(governanceTool).toHaveBeenCalledWith();
  });

  test("passes only the requested Issue name to a chat-bound Issue overview", async () => {
    const issueOverviewTool = vi.fn(async () => "issue_overview");
    vi.mocked(createIssueOverviewTool).mockReturnValue(issueOverviewTool);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      runMode: "read-only",
    });

    await expect(
      executeTool(
        "get_issue_overview",
        { issueName: "projects/sample/issues/42" },
        "tool-issue-overview"
      )
    ).resolves.toEqual({
      kind: "tool_result",
      result: "issue_overview",
    });
    expect(createIssueOverviewTool).toHaveBeenCalledWith({
      projectName: "projects/sample",
    });
    expect(issueOverviewTool).toHaveBeenCalledWith({
      issueName: "projects/sample/issues/42",
    });
  });

  test("passes only the requested database name to a chat-bound database overview", async () => {
    const databaseOverviewTool = vi.fn(async () => "database_overview");
    vi.mocked(createDatabaseOverviewTool).mockReturnValue(databaseOverviewTool);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      runMode: "read-only",
    });

    await expect(
      executeTool(
        "get_database_overview",
        {
          databaseName: "projects/sample/instances/prod/databases/application",
        },
        "tool-database-overview"
      )
    ).resolves.toEqual({
      kind: "tool_result",
      result: "database_overview",
    });
    expect(createDatabaseOverviewTool).toHaveBeenCalledWith({
      projectName: "projects/sample",
    });
    expect(databaseOverviewTool).toHaveBeenCalledWith({
      databaseName: "projects/sample/instances/prod/databases/application",
    });
  });

  test("describes the database backup field as prior-DML-backup readiness", () => {
    const databaseOverview = getToolDefinitions().find(
      (tool) => tool.name === "get_database_overview"
    );

    expect(databaseOverview?.description).toContain(
      "DML prior-backup readiness"
    );
    expect(databaseOverview?.description).toContain(
      "does not inspect existing backups or recovery points"
    );
  });

  test("does not equate an absent review run with review configuration", () => {
    const issueOverview = getToolDefinitions().find(
      (tool) => tool.name === "get_issue_overview"
    );

    expect(issueOverview?.description).toContain(
      "does not establish whether that review is configured or applicable"
    );
  });

  test("uses the chat-bound Project for a starter issue lookup", async () => {
    const issueTool = vi.fn(async () =>
      JSON.stringify({
        project: "projects/sample",
        issues: [
          {
            name: "projects/sample/issues/1",
            title: "Add customer table",
            relation: "review",
            approvalStatus: "PENDING",
          },
        ],
        partial: false,
      })
    );
    vi.mocked(createProjectOpenIssuesTool).mockReturnValue(issueTool);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      runMode: "read-only",
    });

    await expect(
      executeTool("get_project_open_issues", {}, "tool-issues")
    ).resolves.toEqual({
      kind: "tool_result",
      result: expect.stringContaining("Add customer table"),
    });
    expect(createProjectOpenIssuesTool).toHaveBeenCalledWith({
      projectName: "projects/sample",
    });
    expect(issueTool).toHaveBeenCalledWith();
  });

  test("uses the chat-bound Project for a starter recent-activity lookup", async () => {
    const recentActivityTool = vi.fn(async () => "project_recent_activity");
    vi.mocked(createProjectRecentActivityTool).mockReturnValue(
      recentActivityTool
    );
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      runMode: "read-only",
    });

    await expect(
      executeTool("get_project_recent_activity", {}, "tool-recent-activity")
    ).resolves.toEqual({
      kind: "tool_result",
      result: "project_recent_activity",
    });
    expect(createProjectRecentActivityTool).toHaveBeenCalledWith({
      projectName: "projects/sample",
    });
    expect(recentActivityTool).toHaveBeenCalledWith();
  });

  test("uses the chat-bound workspace for a starter recent-activity lookup", async () => {
    const recentActivityTool = vi.fn(async () => "workspace_recent_activity");
    vi.mocked(createWorkspaceRecentActivityTool).mockReturnValue(
      recentActivityTool
    );
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "workspaces/acme",
      runMode: "read-only",
    });

    await expect(
      executeTool("get_workspace_recent_activity", {}, "tool-recent-activity")
    ).resolves.toEqual({
      kind: "tool_result",
      result: "workspace_recent_activity",
    });
    expect(createWorkspaceRecentActivityTool).toHaveBeenCalledWith({
      workspaceName: "workspaces/acme",
    });
    expect(recentActivityTool).toHaveBeenCalledWith();
  });

  test("uses the chat-bound workspace for a starter attention lookup", async () => {
    const attentionTool = vi.fn(async () => "workspace_attention");
    vi.mocked(createWorkspaceAttentionTool).mockReturnValue(attentionTool);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "workspaces/acme",
      runMode: "read-only",
    });

    await expect(
      executeTool("get_workspace_attention", {}, "tool-attention")
    ).resolves.toEqual({
      kind: "tool_result",
      result: "workspace_attention",
    });
    expect(createWorkspaceAttentionTool).toHaveBeenCalledWith({
      workspaceName: "workspaces/acme",
    });
    expect(attentionTool).toHaveBeenCalledWith();
  });

  test("rejects unavailable tools before they can change the page or make a call", async () => {
    const navigateTool = vi.fn(async () => "navigate");
    vi.mocked(createNavigateTool).mockReturnValue(navigateTool);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      runMode: "read-only",
    });
    const callApiMock = vi.mocked(dispatchPreparedApiCall);
    const createDomActionToolMock = vi.mocked(createDomActionTool);
    callApiMock.mockClear();
    createDomActionToolMock.mockClear();

    await expect(
      executeTool("call_api", { operationId: "SQLService/Query" }, "tool-api")
    ).resolves.toEqual({
      kind: "tool_result",
      result: expect.stringContaining(
        "not available during a read-only request"
      ),
    });
    await expect(
      executeTool("dom_action", { type: "click", ref: "e1" }, "tool-dom")
    ).resolves.toEqual({
      kind: "tool_result",
      result: expect.stringContaining(
        "not available during a read-only request"
      ),
    });
    await expect(
      executeTool(
        "navigate",
        { path: "/projects/demo/issues" },
        "tool-navigate"
      )
    ).resolves.toEqual({
      kind: "tool_result",
      result: expect.stringContaining(
        "not available during a read-only request"
      ),
    });

    expect(callApiMock).not.toHaveBeenCalled();
    expect(createDomActionToolMock).not.toHaveBeenCalled();
    expect(navigateTool).not.toHaveBeenCalled();
  });
});

describe("agent tools ask_user", () => {
  test.each(["", " \n\t "])(
    "omits blank confirmation labels %j",
    async (label) => {
      const execute = createToolExecutor({} as AppRouterInstance);
      expect(
        await execute(
          "ask_user",
          {
            kind: "confirm",
            prompt: "Prepare this unsaved draft?",
            confirmLabel: label,
            cancelLabel: label,
          },
          "confirm-blank"
        )
      ).toEqual({
        kind: "ask_user",
        ask: {
          kind: "confirm",
          toolCallId: "confirm-blank",
          prompt: "Prepare this unsaved draft?",
        },
      });
    }
  );

  test("trims nonblank custom confirmation labels", async () => {
    const execute = createToolExecutor({} as AppRouterInstance);
    expect(
      await execute(
        "ask_user",
        {
          kind: "confirm",
          prompt: "Prepare this unsaved draft?",
          confirmLabel: "  Prepare draft  ",
          cancelLabel: "  Not now  ",
        },
        "confirm-custom"
      )
    ).toMatchObject({
      kind: "ask_user",
      ask: { confirmLabel: "Prepare draft", cancelLabel: "Not now" },
    });
  });

  test("rejects a legacy scope purpose on an expiration question and preserves the ordinary retry", async () => {
    const execute = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      workspaceName: "workspaces/example",
    });
    const expiration = {
      kind: "choose",
      resourceKind: "none",
      prompt: "When should this exemption expire?",
      options: [
        { label: "After 24 hours", value: "24-hours" },
        { label: "No expiration", value: "no-expiration" },
      ],
    };

    expect(
      await execute(
        "ask_user",
        { ...expiration, purpose: "masking-exemption-scope" },
        "legacy-expiration"
      )
    ).toMatchObject({
      kind: "tool_result",
      result: expect.stringContaining("ask_masking_exemption_scope"),
    });
    expect(databaseApi.getDatabase).not.toHaveBeenCalled();
    expect(await execute("ask_user", expiration, "expiration")).toEqual({
      kind: "ask_user",
      ask: {
        toolCallId: "expiration",
        kind: "choose",
        prompt: "When should this exemption expire?",
        options: expiration.options,
      },
    });
  });

  describe("database input defaults", () => {
    beforeEach(async () => {
      const actual =
        await vi.importActual<typeof import("./resourceChoices")>(
          "./resourceChoices"
        );
      resourceChoiceApi.validate.mockImplementation(
        actual.validateResourceChoices
      );
      databaseApi.getDatabase.mockImplementation(async ({ name }) =>
        create(DatabaseSchema$, {
          name,
          project: "projects/sample",
          state: State.ACTIVE,
          instanceResource: { title: "Production" },
        })
      );
    });

    test.each([
      { kind: "input", name: "instances/prod/databases/orders" },
      { kind: undefined, name: "instances/prod/databases/orders" },
      {
        kind: "input",
        name: "projects/sample/instances/prod/databases/orders",
      },
    ])(
      "binds a freshly verified $kind default $name",
      async ({ kind, name }) => {
        const executeTool = createToolExecutor({} as AppRouterInstance, {
          context: "projects/sample",
          workspaceName: "workspaces/example",
        });

        expect(
          await executeTool(
            "ask_user",
            {
              kind,
              resourceKind: "database",
              prompt: "Which database?",
              defaultValue: name,
              defaultResource: {
                kind: "database",
                name: "instances/forged/databases/other",
                project: "projects/other",
                label: "Forged label",
              },
            },
            "input-database"
          )
        ).toEqual({
          kind: "ask_user",
          ask: {
            toolCallId: "input-database",
            kind: "input",
            prompt: "Which database?",
            defaultValue: name,
            defaultResource: {
              kind: "database",
              name,
              project: "projects/sample",
              label: "orders (Production)",
            },
          },
        });
      }
    );

    test.each([undefined, "none"])(
      "does not bind an untyped default with resourceKind=%s",
      async (resourceKind) => {
        const executeTool = createToolExecutor({} as AppRouterInstance, {
          context: "projects/sample",
          workspaceName: "workspaces/example",
        });
        expect(
          await executeTool(
            "ask_user",
            {
              resourceKind,
              prompt: "Describe the resource name",
              defaultValue: "instances/prod/databases/orders",
              defaultResource: {
                kind: "database",
                name: "instances/prod/databases/orders",
                project: "projects/sample",
                label: "Forged label",
              },
            },
            "plain-input"
          )
        ).toEqual({
          kind: "ask_user",
          ask: {
            toolCallId: "plain-input",
            kind: "input",
            prompt: "Describe the resource name",
            defaultValue: "instances/prod/databases/orders",
          },
        });
        expect(databaseApi.getDatabase).not.toHaveBeenCalled();
      }
    );

    test.each([undefined, "input"])(
      "allows a database input with kind=%s and no default",
      async (kind) => {
        const executeTool = createToolExecutor({} as AppRouterInstance, {
          context: "workspaces/example",
          workspaceName: "workspaces/example",
        });
        expect(
          await executeTool(
            "ask_user",
            { kind, resourceKind: "database", prompt: "Which database?" },
            "empty-input"
          )
        ).toEqual({
          kind: "ask_user",
          ask: {
            toolCallId: "empty-input",
            kind: "input",
            prompt: "Which database?",
          },
        });
        expect(databaseApi.getDatabase).not.toHaveBeenCalled();
      }
    );

    test("keeps confirmation defaults as plain text", async () => {
      const executeTool = createToolExecutor({} as AppRouterInstance, {
        context: "projects/sample",
        workspaceName: "workspaces/example",
      });
      expect(
        await executeTool(
          "ask_user",
          {
            kind: "confirm",
            resourceKind: "database",
            prompt: "Continue?",
            defaultValue: "instances/prod/databases/orders",
          },
          "confirm-database"
        )
      ).toEqual({
        kind: "ask_user",
        ask: {
          toolCallId: "confirm-database",
          kind: "confirm",
          prompt: "Continue?",
          defaultValue: "instances/prod/databases/orders",
        },
      });
      expect(databaseApi.getDatabase).not.toHaveBeenCalled();
    });

    test.each([
      { context: undefined, workspaceName: "workspaces/example" },
      { context: "workspaces/example", workspaceName: "workspaces/example" },
      { context: "projects/-", workspaceName: "workspaces/example" },
      { context: "projects/sample", workspaceName: undefined },
      { context: "projects/sample", workspaceName: "workspaces/-" },
    ])("rejects a database default without exact scope %j", async (scope) => {
      const executeTool = createToolExecutor({} as AppRouterInstance, scope);
      expect(
        await executeTool(
          "ask_user",
          {
            kind: "input",
            resourceKind: "database",
            prompt: "Which database?",
            defaultValue: "instances/prod/databases/orders",
          },
          "unscoped-input"
        )
      ).toMatchObject({
        kind: "tool_result",
        result: expect.stringContaining("required"),
      });
    });

    test.each([
      "orders",
      "instances/prod/databases/orders/tables/customers",
      " instances/prod/databases/orders ",
      "projects/other/instances/prod/databases/orders",
      null,
      42,
    ])("rejects an invalid database input default %j", async (defaultValue) => {
      const executeTool = createToolExecutor({} as AppRouterInstance, {
        context: "projects/sample",
        workspaceName: "workspaces/example",
      });
      expect(
        await executeTool(
          "ask_user",
          {
            kind: "input",
            resourceKind: "database",
            prompt: "Which database?",
            defaultValue,
          },
          "invalid-input"
        )
      ).toMatchObject({
        kind: "tool_result",
        result: expect.stringContaining("could not be verified"),
      });
    });

    test.each(["denied", "wrong-name", "wrong-project", "deleted"])(
      "rejects a %s database input default using fresh API data",
      async (failure) => {
        databaseApi.getDatabase.mockImplementation(async () => {
          if (failure === "denied") throw new Error("Permission denied");
          return create(DatabaseSchema$, {
            name:
              failure === "wrong-name"
                ? "instances/prod/databases/other"
                : "instances/prod/databases/orders",
            project:
              failure === "wrong-project"
                ? "projects/other"
                : "projects/sample",
            state: failure === "deleted" ? State.DELETED : State.ACTIVE,
          });
        });
        const executeTool = createToolExecutor({} as AppRouterInstance, {
          context: "projects/sample",
          workspaceName: "workspaces/example",
        });
        expect(
          await executeTool(
            "ask_user",
            {
              kind: "input",
              resourceKind: "database",
              prompt: "Which database?",
              defaultValue: "instances/prod/databases/orders",
            },
            "unavailable-input"
          )
        ).toMatchObject({
          kind: "tool_result",
          result: expect.stringContaining("could not be verified"),
        });
      }
    );
  });

  test.each(["projects/sample", "workspaces/example"])(
    "keeps ordinary preferences as choices in %s",
    async (context) => {
      const executeTool = createToolExecutor({} as AppRouterInstance, {
        context,
        workspaceName: "workspaces/example",
      });
      const options = [
        { label: "Fully hide email values", value: "full" },
        { label: "Partially mask email values", value: "partial" },
      ];

      await expect(
        executeTool(
          "ask_user",
          {
            kind: "choose",
            resourceKind: "none",
            prompt: "Should email values be fully hidden or partially masked?",
            options,
          },
          "choose-visibility"
        )
      ).resolves.toEqual({
        kind: "ask_user",
        ask: {
          toolCallId: "choose-visibility",
          kind: "choose",
          prompt: "Should email values be fully hidden or partially masked?",
          options,
        },
      });
      expect(resourceChoiceApi.validate).not.toHaveBeenCalled();
    }
  );

  test.each(["projects/sample", "workspaces/example"])(
    "keeps a mistagged preference retry as a choice in %s",
    async (context) => {
      resourceChoiceApi.validate.mockResolvedValue(undefined);
      const executeTool = createToolExecutor({} as AppRouterInstance, {
        context,
        workspaceName: "workspaces/example",
      });
      const result = await executeTool(
        "ask_user",
        {
          kind: "choose",
          resourceKind: "database",
          prompt: "Should email values be fully hidden or partially masked?",
          options: [{ value: "full" }, { value: "partial" }],
        },
        "choose-visibility"
      );

      expect(result).toMatchObject({
        kind: "tool_result",
        result: expect.stringContaining('resourceKind="none"'),
      });
      if (result.kind === "tool_result") {
        expect(result.result).toContain('kind="choose"');
        expect(result.result).not.toContain("free-form");
      }
    }
  );

  test("rejects mixed database and scope options using the real validator", async () => {
    const actual =
      await vi.importActual<typeof import("./resourceChoices")>(
        "./resourceChoices"
      );
    resourceChoiceApi.validate.mockImplementation(
      actual.validateResourceChoices
    );
    databaseApi.getDatabase.mockResolvedValue(
      create(DatabaseSchema$, {
        name: "instances/prod/databases/orders",
        project: "projects/sample",
        state: State.ACTIVE,
        instanceResource: { title: "Production" },
      })
    );
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      workspaceName: "workspaces/example",
    });
    const result = await executeTool(
      "ask_user",
      {
        kind: "choose",
        resourceKind: "database",
        prompt: "Which database scope?",
        options: [
          { value: "instances/prod/databases/orders" },
          { value: "workspace" },
        ],
      },
      "choose-mixed"
    );

    expect(resourceChoiceApi.validate).toHaveBeenCalledWith(
      "database",
      ["instances/prod/databases/orders", "workspace"],
      "projects/sample",
      "workspaces/example"
    );
    expect(result).toMatchObject({
      kind: "tool_result",
      result: expect.stringContaining("could not be verified"),
    });
    expect(databaseApi.getDatabase).not.toHaveBeenCalled();
  });

  test("keeps scope intents separate from the subsequent verified database picker", async () => {
    const actual =
      await vi.importActual<typeof import("./resourceChoices")>(
        "./resourceChoices"
      );
    resourceChoiceApi.validate.mockImplementation(
      actual.validateResourceChoices
    );
    databaseApi.getDatabase.mockResolvedValue(
      create(DatabaseSchema$, {
        name: "instances/prod/databases/orders",
        project: "projects/sample",
        state: State.ACTIVE,
        instanceResource: { title: "Production" },
      })
    );
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      workspaceName: "workspaces/example",
    });
    const options = [
      { label: "Only the previously queried database", value: "this_database" },
      { label: "All databases in this project", value: "project" },
      {
        label: "Choose another database in this project",
        value: "another_database",
      },
    ];
    expect(
      await executeTool(
        "ask_user",
        {
          kind: "choose",
          resourceKind: "none",
          prompt: "Which database scope should the global rule cover?",
          options,
        },
        "choose-scope"
      )
    ).toEqual({
      kind: "ask_user",
      ask: {
        toolCallId: "choose-scope",
        kind: "choose",
        prompt: "Which database scope should the global rule cover?",
        options,
      },
    });
    expect(databaseApi.getDatabase).not.toHaveBeenCalled();
    expect(
      await executeTool(
        "ask_user",
        {
          kind: "choose",
          resourceKind: "database",
          prompt: "Which database?",
          options: [
            {
              label: "Unverified label",
              value: "instances/prod/databases/orders",
            },
          ],
        },
        "choose-database"
      )
    ).toEqual({
      kind: "ask_user",
      ask: {
        toolCallId: "choose-database",
        kind: "choose",
        prompt: "Which database?",
        options: [
          {
            label: "orders (Production)",
            value: "instances/prod/databases/orders",
          },
        ],
      },
    });
    expect(databaseApi.getDatabase).toHaveBeenCalledOnce();
  });

  describe("resumed masking database choices", () => {
    const database = "instances/prod/databases/orders";
    const pickerArgs = (kind: "choose" | "input") => ({
      kind,
      resourceKind: "database",
      prompt: "Which database?",
      ...(kind === "choose"
        ? { options: [{ label: "Unverified label", value: database }] }
        : { defaultValue: database }),
    });

    beforeEach(async () => {
      const actual =
        await vi.importActual<typeof import("./resourceChoices")>(
          "./resourceChoices"
        );
      resourceChoiceApi.validate.mockImplementation(
        actual.validateResourceChoices
      );
      databaseApi.getDatabase.mockResolvedValue(
        create(DatabaseSchema$, {
          name: database,
          project: "projects/sample",
          state: State.ACTIVE,
          instanceResource: { title: "Production" },
        })
      );
    });

    test.each(["choose", "input"] as const)(
      "uses the selected project for a workspace chat's database %s",
      async (kind) => {
        const execute = createToolExecutor({} as AppRouterInstance, {
          context: "workspaces/example",
          workspaceName: "workspaces/example",
          databaseChoiceProject: "projects/sample",
        });
        const result = await execute("ask_user", pickerArgs(kind), "database");

        expect(result).toMatchObject({
          kind: "ask_user",
          ask:
            kind === "choose"
              ? { options: [{ label: "orders (Production)", value: database }] }
              : {
                  defaultResource: {
                    kind: "database",
                    name: database,
                    project: "projects/sample",
                    label: "orders (Production)",
                  },
                },
        });
      }
    );

    test.each([
      undefined,
      "projects/-",
      "projects/sample/databases/orders",
      "projects/invalid project",
    ])(
      "does not use an invalid selected project %j",
      async (databaseChoiceProject) => {
        const execute = createToolExecutor({} as AppRouterInstance, {
          context: "workspaces/example",
          workspaceName: "workspaces/example",
          databaseChoiceProject,
        });
        for (const kind of ["choose", "input"] as const) {
          expect(
            await execute("ask_user", pickerArgs(kind), `database-${kind}`)
          ).toMatchObject({
            kind: "tool_result",
            result: expect.stringContaining("required"),
          });
        }
        expect(databaseApi.getDatabase).not.toHaveBeenCalled();
      }
    );

    test.each(["different-project", "inactive", "unavailable"])(
      "revalidates selected-project databases after they become %s",
      async (change) => {
        if (change === "unavailable") {
          databaseApi.getDatabase.mockRejectedValue(
            new Error("Access revoked")
          );
        } else {
          databaseApi.getDatabase.mockResolvedValue(
            create(DatabaseSchema$, {
              name: database,
              project:
                change === "different-project"
                  ? "projects/other"
                  : "projects/sample",
              state: change === "inactive" ? State.DELETED : State.ACTIVE,
            })
          );
        }
        const execute = createToolExecutor({} as AppRouterInstance, {
          context: "workspaces/example",
          workspaceName: "workspaces/example",
          databaseChoiceProject: "projects/sample",
        });
        for (const kind of ["choose", "input"] as const) {
          expect(
            await execute("ask_user", pickerArgs(kind), `database-${kind}`)
          ).toMatchObject({
            kind: "tool_result",
            result: expect.stringContaining("could not be verified"),
          });
        }
        expect(databaseApi.getDatabase).toHaveBeenCalledTimes(2);
      }
    );

    test.each([
      { context: "projects/sample" },
      { context: "workspaces/example", referencedProject: "projects/sample" },
    ])(
      "retains the bound project instead of a conflicting selection %j",
      async (scope) => {
        const execute = createToolExecutor({} as AppRouterInstance, {
          ...scope,
          workspaceName: "workspaces/example",
          databaseChoiceProject: "projects/other",
        });
        expect(
          await execute("ask_user", pickerArgs("choose"), "database")
        ).toMatchObject({
          kind: "ask_user",
          ask: { options: [{ value: database }] },
        });
        databaseApi.getDatabase.mockResolvedValue(
          create(DatabaseSchema$, {
            name: database,
            project: "projects/other",
            state: State.ACTIVE,
          })
        );
        expect(
          await execute("ask_user", pickerArgs("choose"), "foreign-database")
        ).toMatchObject({
          kind: "tool_result",
          result: expect.stringContaining("could not be verified"),
        });
      }
    );

    test("does not narrow a workspace rule to the database picker's project", async () => {
      const execute = createToolExecutor({} as AppRouterInstance, {
        context: "workspaces/example",
        workspaceName: "workspaces/example",
        databaseChoiceProject: "projects/sample",
        userName: "users/member@example.com",
        chatId: "chat",
      });
      expect(
        await execute(
          "prepare_masking_rule_draft",
          {
            title: "Mask email",
            semanticType: "bb.default",
            condition: 'resource.column_name == "email"',
            scope: { kind: "workspace" },
          },
          "rule"
        )
      ).toMatchObject({
        kind: "tool_result",
        result: expect.stringContaining('"draftId":"draft-1"'),
      });
    });
  });

  describe("masking exemption scope choices", () => {
    beforeEach(async () => {
      const actual =
        await vi.importActual<typeof import("./resourceChoices")>(
          "./resourceChoices"
        );
      resourceChoiceApi.validate.mockImplementation(
        actual.validateResourceChoices
      );
      databaseApi.getDatabase.mockResolvedValue(
        create(DatabaseSchema$, {
          name: "instances/prod/databases/orders",
          project: "projects/sample",
          state: State.ACTIVE,
          instanceResource: { title: "Production" },
        })
      );
    });

    test("supplies complete unselected scope choices with the verified database label", async () => {
      const execute = createToolExecutor({} as AppRouterInstance, {
        context: "projects/sample",
        workspaceName: "workspaces/example",
        historicalDatabase: {
          database: "instances/prod/databases/orders",
          project: "projects/sample",
        },
      });
      expect(await execute("ask_masking_exemption_scope", {}, "scope")).toEqual(
        {
          kind: "ask_user",
          ask: {
            toolCallId: "scope",
            kind: "choose",
            prompt: "Which databases should this exemption cover?",
            options: [
              {
                label: "Only orders (Production)",
                value: "instances/prod/databases/orders",
              },
              {
                label: "All databases in this project",
                value: "projects/sample",
              },
              { label: "Another database", value: "another-database" },
            ],
          },
        }
      );
    });

    test.each(["denied", "moved", "deleted", "renamed"])(
      "does not offer a %s historical database",
      async (state) => {
        if (state === "denied")
          databaseApi.getDatabase.mockRejectedValue(new Error("Forbidden"));
        else
          databaseApi.getDatabase.mockResolvedValue(
            create(DatabaseSchema$, {
              name:
                state === "renamed"
                  ? "instances/prod/databases/other"
                  : "instances/prod/databases/orders",
              project: state === "moved" ? "projects/other" : "projects/sample",
              state: state === "deleted" ? State.DELETED : State.ACTIVE,
            })
          );
        const execute = createToolExecutor({} as AppRouterInstance, {
          context: "projects/sample",
          workspaceName: "workspaces/example",
          historicalDatabase: {
            database: "instances/prod/databases/orders",
            project: "projects/sample",
          },
        });
        const result = await execute(
          "ask_masking_exemption_scope",
          {},
          "scope"
        );
        expect(result).toMatchObject({
          kind: "ask_user",
          ask: {
            options: [
              {
                label: "All databases in this project",
                value: "projects/sample",
              },
              { label: "Another database", value: "another-database" },
            ],
          },
        });
      }
    );

    test.each([
      { context: "workspaces/example", workspaceName: "workspaces/example" },
      { context: "projects/-", workspaceName: "workspaces/example" },
      { context: "projects/sample", workspaceName: "workspaces/-" },
      { context: "projects/sample", workspaceName: undefined },
    ])("requires an exact Project and Workspace: %j", async (scope) => {
      const execute = createToolExecutor({} as AppRouterInstance, scope);
      expect(
        await execute("ask_masking_exemption_scope", {}, "scope")
      ).toMatchObject({
        kind: "tool_result",
        result: expect.stringContaining("exact Project and Workspace"),
      });
      expect(databaseApi.getDatabase).not.toHaveBeenCalled();
    });

    test("ignores a historical candidate owned by another project", async () => {
      const execute = createToolExecutor({} as AppRouterInstance, {
        context: "projects/sample",
        workspaceName: "workspaces/example",
        historicalDatabase: {
          database: "instances/prod/databases/orders",
          project: "projects/other",
        },
      });
      expect(
        await execute("ask_masking_exemption_scope", {}, "scope")
      ).toMatchObject({
        kind: "ask_user",
        ask: {
          options: [
            { value: "projects/sample" },
            { value: "another-database" },
          ],
        },
      });
      expect(databaseApi.getDatabase).not.toHaveBeenCalled();
    });

    test("uses the explicit database reference before model and historical suggestions", async () => {
      const execute = createToolExecutor({} as AppRouterInstance, {
        context: "projects/sample",
        workspaceName: "workspaces/example",
        referencedDatabase: "instances/prod/databases/orders",
        historicalDatabase: {
          database: "instances/prod/databases/old",
          project: "projects/sample",
        },
      });
      expect(
        await execute(
          "ask_masking_exemption_scope",
          {
            database: "instances/prod/databases/other",
          },
          "scope"
        )
      ).toMatchObject({
        kind: "ask_user",
        ask: {
          options: [
            {
              label: "Only orders (Production)",
              value: "instances/prod/databases/orders",
            },
            { value: "projects/sample" },
            { value: "another-database" },
          ],
        },
      });
      expect(resourceChoiceApi.validate).toHaveBeenCalledWith(
        "database",
        ["instances/prod/databases/orders"],
        "projects/sample",
        "workspaces/example"
      );
    });

    test("verifies an exact database candidate supplied with the scope question", async () => {
      const execute = createToolExecutor({} as AppRouterInstance, {
        context: "projects/sample",
        workspaceName: "workspaces/example",
      });
      expect(
        await execute(
          "ask_masking_exemption_scope",
          {
            database: "instances/prod/databases/orders",
          },
          "scope"
        )
      ).toMatchObject({
        kind: "ask_user",
        ask: {
          options: [
            {
              label: "Only orders (Production)",
              value: "instances/prod/databases/orders",
            },
            { value: "projects/sample" },
            { value: "another-database" },
          ],
        },
      });
    });

    test("localizes the code-owned question and options in every supported language", async () => {
      try {
        for (const language of ["en-US", "zh-CN", "es-ES", "ja-JP", "vi-VN"]) {
          await i18n.changeLanguage(language);
          const execute = createToolExecutor({} as AppRouterInstance, {
            context: "projects/sample",
            workspaceName: "workspaces/example",
          });
          const result = await execute(
            "ask_masking_exemption_scope",
            {},
            "scope"
          );
          expect(result.kind).toBe("ask_user");
          if (result.kind !== "ask_user")
            throw new Error("Expected scope choices");
          const question = i18n.getResource(
            language,
            "translation",
            "agent.masking.exemption-scope.question"
          );
          expect(question).toEqual(expect.any(String));
          expect(result.ask.prompt).toBe(question);
          for (const [index, key] of [
            "project",
            "another-database",
          ].entries()) {
            const translation = i18n.getResource(
              language,
              "translation",
              `agent.masking.exemption-scope.${key}`
            );
            expect(translation).toEqual(expect.any(String));
            expect(result.ask.options?.[index].label).toBe(translation);
          }
          expect(
            i18n.getResource(
              language,
              "translation",
              "agent.masking.exemption-scope.only-database"
            )
          ).toContain("{{database}}");
        }
      } finally {
        await i18n.changeLanguage("en-US");
      }
    });

    test.each([
      { prompt: "When should this exemption expire?" },
      { kind: "input" },
      { options: [{ label: "No expiration", value: "no-expiration" }] },
      { defaultValue: "projects/sample" },
      { purpose: "masking-exemption-scope" },
      { resourceKind: "none" },
      { project: "projects/other" },
    ])("rejects model-owned question fields %j", async (extra) => {
      const execute = createToolExecutor({} as AppRouterInstance, {
        context: "projects/sample",
        workspaceName: "workspaces/example",
      });
      expect(
        await execute("ask_masking_exemption_scope", extra, "scope")
      ).toMatchObject({
        kind: "tool_result",
        result: expect.stringMatching(/error/i),
      });
      expect(databaseApi.getDatabase).not.toHaveBeenCalled();
    });

    test("separates the fixed scope tool schema from ordinary questions", () => {
      const definitions = getToolDefinitions({
        maskingSections: ["exemption"],
      });
      const generic = definitions.find((tool) => tool.name === "ask_user");
      const scope = definitions.find(
        (tool) => tool.name === "ask_masking_exemption_scope"
      );

      expect(generic?.parametersSchema.properties).not.toHaveProperty(
        "purpose"
      );
      expect(generic?.parametersSchema.properties).not.toHaveProperty(
        "database"
      );
      expect(scope?.parametersSchema).toEqual({
        type: "object",
        properties: {
          database: expect.objectContaining({ type: "string" }),
        },
        additionalProperties: false,
      });
    });
  });

  test("replaces model labels with verified resource choices", async () => {
    resourceChoiceApi.validate.mockResolvedValue([
      {
        label: "orders (Production)",
        value: "instances/prod/databases/orders",
      },
    ]);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      workspaceName: "workspaces/example",
    });
    const result = await executeTool(
      "ask_user",
      {
        kind: "choose",
        resourceKind: "database",
        prompt: "Which database?",
        defaultValue: "instances/prod/databases/orders",
        options: [
          { label: "Forged label", value: "instances/prod/databases/orders" },
        ],
      },
      "choose-1"
    );
    expect(resourceChoiceApi.validate).toHaveBeenCalledWith(
      "database",
      ["instances/prod/databases/orders"],
      "projects/sample",
      "workspaces/example"
    );
    expect(result).toEqual({
      kind: "ask_user",
      ask: {
        toolCallId: "choose-1",
        prompt: "Which database?",
        kind: "choose",
        defaultValue: "instances/prod/databases/orders",
        options: [
          {
            label: "orders (Production)",
            value: "instances/prod/databases/orders",
          },
        ],
      },
    });
  });

  test("asks for a Project before offering database choices", async () => {
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "workspaces/example",
      workspaceName: "workspaces/example",
    });
    const result = await executeTool(
      "ask_user",
      {
        kind: "choose",
        resourceKind: "database",
        prompt: "Which database?",
        options: [{ value: "instances/prod/databases/orders" }],
      },
      "choose-1"
    );
    expect(result.kind).toBe("tool_result");
    expect(result).toEqual({
      kind: "tool_result",
      result: expect.stringContaining("A Project is required"),
    });
    expect(resourceChoiceApi.validate).not.toHaveBeenCalled();
  });

  test("does not offer a dedicated environment resource choice", async () => {
    const askUser = getToolDefinitions().find(
      (tool) => tool.name === "ask_user"
    );
    expect(askUser?.parametersSchema).toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          resourceKind: expect.objectContaining({
            enum: ["none", "database"],
          }),
        }),
      })
    );

    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      workspaceName: "workspaces/example",
    });
    const result = await executeTool(
      "ask_user",
      {
        kind: "choose",
        resourceKind: "environment",
        prompt: "Which environment?",
        options: [{ value: "environments/test" }],
      },
      "choose-environment"
    );
    expect(result).toEqual({
      kind: "tool_result",
      result: expect.stringContaining("Unsupported resource choice kind"),
    });
    expect(resourceChoiceApi.validate).not.toHaveBeenCalled();
  });

  test("does not pause for unverified resource options", async () => {
    resourceChoiceApi.validate.mockResolvedValue(undefined);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      context: "projects/sample",
      workspaceName: "workspaces/example",
    });
    const result = await executeTool(
      "ask_user",
      {
        kind: "choose",
        resourceKind: "database",
        prompt: "Which database?",
        options: [{ value: "instances/prod/databases/orders" }],
      },
      "choose-1"
    );
    expect(result).toEqual({
      kind: "tool_result",
      result: expect.stringContaining("could not be verified"),
    });
    if (result.kind === "tool_result") {
      expect(result.result).toContain('kind="choose"');
      expect(result.result).not.toContain("free-form");
    }
  });

  test("accepts a legacy typed handoff on done", async () => {
    const executeTool = createToolExecutor({} as AppRouterInstance);
    await expect(
      executeTool(
        "done",
        {
          text: "Review the issue queue.",
          success: true,
          handoff: {
            kind: "review-project-issues",
            project: "projects/sample",
          },
        },
        "done-1"
      )
    ).resolves.toEqual({
      kind: "done",
      text: "Review the issue queue.",
      success: true,
      handoff: { kind: "review-project-issues", project: "projects/sample" },
    });
  });

  test("accepts typed Open and Continue proposals on done", async () => {
    const definition = getToolDefinitions().find(
      (tool) => tool.name === "done"
    );
    expect(definition?.description).toContain(
      "up to three distinct contextual Continue questions"
    );
    expect(definition?.parametersSchema).toMatchObject({
      properties: { actions: { type: "array", maxItems: 4 } },
      required: ["text", "success", "actions"],
    });
    const executeTool = createToolExecutor({} as AppRouterInstance);
    await expect(
      executeTool(
        "done",
        {
          text: "Review the issue queue.",
          success: true,
          actions: [
            {
              kind: "open",
              target: {
                kind: "review-project-issues",
                project: "projects/sample",
              },
            },
            { kind: "continue", prompt: "  What is blocked?  " },
            { kind: "continue", prompt: "Who should review it?" },
            { kind: "continue", prompt: "How can I verify it?" },
          ],
        },
        "done-actions"
      )
    ).resolves.toEqual({
      kind: "done",
      text: "Review the issue queue.",
      success: true,
      actions: [
        {
          kind: "open",
          target: {
            kind: "review-project-issues",
            project: "projects/sample",
          },
        },
        { kind: "continue", prompt: "What is blocked?" },
        { kind: "continue", prompt: "Who should review it?" },
        { kind: "continue", prompt: "How can I verify it?" },
      ],
    });
  });

  test("exposes exact Issue, Plan, and failed Task Open targets", () => {
    const definition = getToolDefinitions().find(
      (tool) => tool.name === "done"
    );
    expect(definition?.parametersSchema).toMatchObject({
      properties: {
        actions: {
          items: {
            properties: {
              target: {
                properties: {
                  kind: {
                    enum: expect.arrayContaining([
                      "open-issue",
                      "review-issue-plan",
                      "view-failed-task",
                    ]),
                  },
                  issue: { type: "string" },
                  plan: { type: "string" },
                  task: { type: "string" },
                },
              },
            },
          },
        },
      },
    });
  });

  test("offers a typed draft Plan action without putting SQL in the action", async () => {
    const definition = getToolDefinitions().find(
      (tool) => tool.name === "done"
    );
    expect(definition?.parametersSchema).toMatchObject({
      properties: {
        actions: {
          items: {
            properties: {
              kind: {
                enum: [
                  "open",
                  "draft-plan",
                  "draft-issue-comment",
                  "draft-masking-rule",
                  "draft-masking-exemption",
                  "draft-semantic-type",
                  "continue",
                ],
              },
              project: { type: "string" },
              database: { type: "string" },
            },
          },
        },
      },
    });
    expect(definition?.parametersSchema).not.toHaveProperty(
      "properties.actions.items.properties.sql"
    );

    const executeTool = createToolExecutor({} as AppRouterInstance);
    await expect(
      executeTool(
        "done",
        {
          text: "```sql\nCREATE TABLE audit_log (id INT);\n```",
          success: true,
          actions: [
            {
              kind: "draft-plan",
              project: "projects/sample",
              database: "instances/prod/databases/app",
            },
          ],
        },
        "done-plan-draft"
      )
    ).resolves.toMatchObject({
      kind: "done",
      success: true,
      actions: [
        {
          kind: "draft-plan",
          project: "projects/sample",
          database: "instances/prod/databases/app",
        },
      ],
    });
  });

  test("describes masking draft actions as runtime-owned member handoffs", () => {
    const definitions = getToolDefinitions({
      maskingSections: ["rule", "semantic-type"],
    });
    const guidance = getMaskingSkillGuidance(["rule", "semantic-type"]);
    const done = definitions.find((tool) => tool.name === "done");
    const prepare = definitions.find(
      (tool) => tool.name === "prepare_masking_rule_draft"
    );

    expect(done?.description).toContain(
      "Runtime supplies eligible prepared masking actions"
    );
    expect(guidance).toContain("draft-masking-rule");
    expect(guidance).toContain("draft-semantic-type");
    expect(guidance).toContain(
      "runtime supplies the latest eligible current-run masking draft action"
    );
    expect(done?.parametersSchema).toMatchObject({
      properties: {
        actions: {
          description: expect.stringContaining(
            "draft-masking-rule opens or applies an unsaved masking form only after a member click"
          ),
          items: {
            properties: {
              draftId: {
                description: expect.stringContaining(
                  "Normally omit the entire draft-masking-rule action"
                ),
              },
            },
          },
        },
      },
    });
    expect(prepare?.description).toContain("No opening or saving occurs");
    expect(guidance).toContain("you do not need to repeat its opaque draft ID");
    expect(guidance).toContain(
      "After a resumed run or source change, read the current form and prepare again"
    );
  });

  test("defers masking catalog and scope choices to permission-aware runtime tools", () => {
    const definitions = getToolDefinitions({ maskingSections: ["rule"] });
    const ask = definitions.find((tool) => tool.name === "ask_user");
    const guidance = getMaskingSkillGuidance(["rule"]);
    expect(ask?.description).toContain(
      "dedicated scope/type pickers, not fabricated alternatives"
    );
    expect(guidance).toContain("ask_masking_semantic_type");
    expect(guidance).toContain("ask_masking_rule_scope");
    expect(definitions.map((tool) => tool.name)).toEqual(
      expect.arrayContaining([
        "ask_masking_semantic_type",
        "ask_masking_rule_scope",
      ])
    );
    expect(ask?.description).not.toContain(
      "ordinary preferences such as full versus partial masking"
    );
  });

  test("keeps existing-rule snapshots review-only in masking draft tool guidance", () => {
    const definitions = getToolDefinitions({ maskingSections: ["rule"] });
    const guidance = getMaskingSkillGuidance(["rule"]);
    const read = definitions.find(
      (tool) => tool.name === "get_masking_rule_draft"
    );
    const prepare = definitions.find(
      (tool) => tool.name === "prepare_masking_rule_draft"
    );

    expect(read?.description).toContain("EDIT is review-only");
    expect(prepare?.description).toContain("refine its CREATE editor");
    expect(guidance).toContain(
      "Only CREATE supports full-rule preparation and Apply to draft"
    );
    expect(guidance).toContain(
      "Never prepare or apply a full-rule replacement in EDIT"
    );
  });

  test.each(["done", "prepare_masking_rule_draft", "get_masking_overview"])(
    "pairs %s with loaded draft-first masking setup guidance",
    (name) => {
      const definition = getToolDefinitions({ maskingSections: ["rule"] }).find(
        (tool) => tool.name === name
      );
      expect(definition).toBeDefined();
      const guidance = getMaskingSkillGuidance(["rule"]);
      expect(guidance).toContain(
        "MUST call prepare_masking_rule_draft before done"
      );
      expect(guidance).toContain(
        "Do not finish a ready setup with only manual steps or an Open settings action"
      );
      expect(guidance).toContain(
        "Conceptual or effect questions, explicitly requested manual-only instructions"
      );
    }
  );

  test("exposes an unsent Issue comment draft action", () => {
    const definition = getToolDefinitions().find(
      (tool) => tool.name === "done"
    );
    expect(definition?.parametersSchema).toMatchObject({
      properties: {
        actions: {
          items: {
            properties: {
              issue: { type: "string" },
              comment: { type: "string" },
            },
          },
        },
      },
    });
  });

  test("drops a malformed legacy handoff", async () => {
    const executeTool = createToolExecutor({} as AppRouterInstance);
    await expect(
      executeTool(
        "done",
        {
          text: "Open the project.",
          success: true,
          handoff: {
            kind: "open-project-instances",
            project: "projects/sample",
            href: "https://example.com",
          },
        },
        "done-1"
      )
    ).resolves.toEqual({
      kind: "done",
      text: "Open the project.",
      success: true,
    });
  });

  test("exposes choose in the ask_user schema", () => {
    const askUser = getToolDefinitions().find(
      (tool) => tool.name === "ask_user"
    );

    expect(askUser).toBeDefined();
    expect(askUser?.parametersSchema).toEqual(
      expect.objectContaining({
        properties: expect.objectContaining({
          kind: expect.objectContaining({
            enum: ["input", "confirm", "choose"],
          }),
          options: expect.any(Object),
        }),
      })
    );
  });

  test("parses choose prompts with sanitized options", async () => {
    const executeTool = createToolExecutor({} as AppRouterInstance);

    const result = await executeTool(
      "ask_user",
      {
        prompt: "Which environment should I use?",
        kind: "choose",
        options: [
          {
            label: " Production ",
            value: " prod ",
            description: " Primary ",
          },
          {
            value: "staging",
          },
          {
            label: "Broken",
          },
        ],
      },
      "tool-choose"
    );

    expect(result).toEqual({
      kind: "ask_user",
      ask: {
        toolCallId: "tool-choose",
        prompt: "Which environment should I use?",
        kind: "choose",
        options: [
          {
            label: "Production",
            value: "prod",
            description: "Primary",
          },
          {
            label: "staging",
            value: "staging",
          },
        ],
      },
    });
  });

  test("requests corrected choices when choose options are missing", async () => {
    const executeTool = createToolExecutor({} as AppRouterInstance);

    await expect(
      executeTool(
        "ask_user",
        {
          prompt: "Which environment should I use?",
          kind: "choose",
        },
        "tool-choose"
      )
    ).rejects.toThrow('Retry with kind="choose"');
  });

  test("requests distinct choice values instead of ambiguous options", async () => {
    const executeTool = createToolExecutor({} as AppRouterInstance);

    await expect(
      executeTool(
        "ask_user",
        {
          prompt: "How should emails be masked?",
          kind: "choose",
          options: [
            { label: "Fully hidden", value: "full" },
            { label: "Partially visible", value: " full " },
          ],
        },
        "tool-choose"
      )
    ).rejects.toThrow('Retry with kind="choose" and distinct option values');
  });
});

describe("agent tools search_api", () => {
  test("renders byte fields as bytes with the UTF-8 call_api note", async () => {
    const { searchApi: actualSearchApi } =
      await vi.importActual<typeof import("./searchApi")>("./searchApi");

    const result = await actualSearchApi({
      operationId: "SheetService/CreateSheet",
    });

    const requestBodyStart = result.indexOf("### Request Body");
    const responseBodyStart = result.indexOf("### Response Body");
    const requestBody =
      requestBodyStart >= 0
        ? result.slice(
            requestBodyStart,
            responseBodyStart >= 0 ? responseBodyStart : undefined
          )
        : "";
    const responseBody =
      responseBodyStart >= 0 ? result.slice(responseBodyStart) : "";
    const requestNote =
      "request body includes protobuf bytes fields; plain strings passed to call_api are UTF-8 encoded automatically.";
    const requestNoteCount = requestBody.split(requestNote).length - 1;

    expect(requestBody).toContain('"sheet": bytebase.v1.Sheet');
    expect(requestBody).toContain(requestNote);
    expect(requestNoteCount).toBe(1);
    expect(responseBody).toContain('"content": bytes');
    expect(responseBody).not.toContain(requestNote);
  });

  test("detects transitive byte fields through external oneof request schemas", async () => {
    const { searchApi: actualSearchApi } =
      await vi.importActual<typeof import("./searchApi")>("./searchApi");

    const result = await actualSearchApi({
      operationId: "CelService/BatchDeparse",
    });

    expect(result).toContain(
      '"expressions": array<google.api.expr.v1alpha1.Expr>'
    );
    expect(result).toContain(
      "Note: request body includes protobuf bytes fields; plain strings passed to call_api are UTF-8 encoded automatically."
    );
  });
});

describe("agent tools concurrency guard", () => {
  test.each(["get_page_state", "navigate", "dom_action"])(
    "does not use the live page for background %s calls",
    async (name) => {
      const pageState = vi.fn(async () => "page_state");
      const navigate = vi.fn(async () => "navigate");
      const domAction = vi.fn(async () => "dom_action");
      vi.mocked(createPageStateTool).mockReturnValue(pageState);
      vi.mocked(createNavigateTool).mockReturnValue(navigate);
      vi.mocked(createDomActionTool).mockReturnValue(domAction);

      const executeTool = createToolExecutor({} as AppRouterInstance, {
        chatId: "background-chat",
        canUseCurrentPage: () => false,
      });
      const result = await executeTool(name, {}, `tool-${name}`);

      expect(result).toMatchObject({
        kind: "tool_result",
        result: expect.stringContaining("not selected"),
      });
      expect(pageState).not.toHaveBeenCalled();
      expect(navigate).not.toHaveBeenCalled();
      expect(domAction).not.toHaveBeenCalled();
    }
  );

  test("resumes page tools after reselecting a chat without blocking resource tools", async () => {
    let selected = false;
    const canUseCurrentPage = () => selected;
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      chatId: "background-chat",
      canUseCurrentPage,
    });
    expect(createDomActionTool).toHaveBeenLastCalledWith(
      expect.anything(),
      canUseCurrentPage
    );

    expect(
      await executeTool("get_project_overview", {}, "tool-overview")
    ).toEqual({
      kind: "tool_result",
      result: "project_overview",
    });
    expect(await executeTool("search_api", {}, "tool-search")).toEqual({
      kind: "tool_result",
      result: "search_api",
    });
    expect(
      await executeTool("get_page_state", {}, "tool-page-1")
    ).toMatchObject({
      kind: "tool_result",
      result: expect.stringContaining("not selected"),
    });

    selected = true;
    expect(await executeTool("get_page_state", {}, "tool-page-2")).toEqual({
      kind: "tool_result",
      result: "page_state",
    });
  });

  test("discards a page-state result if its chat is backgrounded during the read", async () => {
    let selected = true;
    let resolveRead: ((result: string) => void) | undefined;
    const pageState = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          resolveRead = resolve;
        })
    );
    vi.mocked(createPageStateTool).mockReturnValue(pageState);
    const executeTool = createToolExecutor({} as AppRouterInstance, {
      chatId: "background-chat",
      canUseCurrentPage: () => selected,
    });

    const result = executeTool("get_page_state", {}, "tool-page");
    expect(pageState).toHaveBeenCalledTimes(1);
    selected = false;
    resolveRead?.("another chat's page");

    await expect(result).resolves.toMatchObject({
      kind: "tool_result",
      result: expect.stringContaining("not selected"),
    });
  });

  test("blocks concurrent page-changing tools across threads", async () => {
    let releaseNavigation: (() => void) | undefined;
    vi.mocked(createNavigateTool).mockImplementation(() => {
      return vi.fn(
        () =>
          new Promise<string>((resolve) => {
            releaseNavigation = () =>
              resolve(
                JSON.stringify({
                  navigated: true,
                  currentPath: "/projects/demo",
                })
              );
          })
      );
    });

    const firstExecutor = createToolExecutor({} as AppRouterInstance, {
      chatId: "thread-1",
    });
    const secondExecutor = createToolExecutor({} as AppRouterInstance, {
      chatId: "thread-2",
    });

    const firstRun = firstExecutor(
      "navigate",
      { path: "/projects/demo" },
      "tool-navigate-1"
    );

    await expect(
      secondExecutor("navigate", { path: "/projects/other" }, "tool-navigate-2")
    ).rejects.toThrow("Another chat is already using a page-changing tool");

    releaseNavigation?.();
    await expect(firstRun).resolves.toEqual({
      kind: "tool_result",
      result: JSON.stringify({
        navigated: true,
        currentPath: "/projects/demo",
      }),
    });
  });
});
