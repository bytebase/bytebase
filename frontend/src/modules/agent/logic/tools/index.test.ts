// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { AppRouterInstance } from "@/app/router";
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
import {
  ApprovalFlowSchema,
  ApprovalTemplateSchema,
  Issue_Type,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";

const projectApi = vi.hoisted(() => ({ getProject: vi.fn() }));
const issueApi = vi.hoisted(() => ({ getIssue: vi.fn() }));
const databaseApi = vi.hoisted(() => ({ getDatabase: vi.fn() }));
const celApi = vi.hoisted(() => ({ batchParse: vi.fn() }));

vi.mock("@/api", () => ({
  projectServiceClientConnect: { getProject: projectApi.getProject },
  issueServiceClientConnect: { getIssue: issueApi.getIssue },
  databaseServiceClientConnect: { getDatabase: databaseApi.getDatabase },
  celServiceClientConnect: { batchParse: celApi.batchParse },
}));

vi.mock("../skills", () => ({
  getSkill: vi.fn(async () => "skill"),
}));
vi.mock("./callApi", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./callApi")>()),
  dispatchPreparedApiCall: vi.fn(async () => "call_api"),
}));
vi.mock("./apiPermissionPreflight", () => ({
  checkPreparedApiPermission: vi.fn(async () => ({ kind: "allowed" })),
}));
vi.mock("./databaseCount", () => ({
  createDatabaseCountTool: vi.fn(() => vi.fn(async () => "database_count")),
}));
vi.mock("./databaseOverview", () => ({
  createDatabaseOverviewTool: vi.fn(() =>
    vi.fn(async () => "database_overview")
  ),
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
import { getPendingApiCallPreview } from "./pendingApiCall";
import { createProjectGovernanceOverviewTool } from "./projectGovernanceOverview";
import { createProjectOpenIssuesTool } from "./projectOpenIssues";
import { createProjectOverviewTool } from "./projectOverview";
import { createProjectRecentActivityTool } from "./projectRecentActivity";
import { createWorkspaceAttentionTool } from "./workspaceAttention";
import { createWorkspaceOverviewTool } from "./workspaceOverview";
import { createWorkspaceRecentActivityTool } from "./workspaceRecentActivity";

beforeEach(() => {
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

describe("done tool", () => {
  test("requires an explicit task success value", async () => {
    const definition = getToolDefinitions().find((tool) => tool.name === "done");
    expect(definition?.parametersSchema).toMatchObject({
      required: ["text", "success"],
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

  test("only exposes page inspection, guidance, and completion", () => {
    expect(
      getToolDefinitions({ runMode: "read-only" }).map((tool) => tool.name)
    ).toEqual([
      "get_page_state",
      "get_project_overview",
      "get_project_governance_overview",
      "get_issue_overview",
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
      "get_database_overview"
    );
    expect(getToolDefinitions().map((tool) => tool.name)).toContain(
      "get_database_count"
    );
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

  test("falls back to input when choose options are missing", async () => {
    const executeTool = createToolExecutor({} as AppRouterInstance);

    const result = await executeTool(
      "ask_user",
      {
        prompt: "Which environment should I use?",
        kind: "choose",
      },
      "tool-choose"
    );

    expect(result).toEqual({
      kind: "ask_user",
      ask: {
        toolCallId: "tool-choose",
        prompt: "Which environment should I use?",
        kind: "input",
      },
    });
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
