// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import {
  ApprovalStatus,
  IssueStatus,
  State,
} from "@/types/proto-es/v1/common_pb";
import { type Issue, Issue_Type } from "@/types/proto-es/v1/issue_service_pb";

const state = {
  currentUser: { name: "users/ed@bytebase.com" },
  serverInfo: { defaultProject: "projects/default" },
  workspacePolicy: { bindings: [] },
  projectPoliciesByName: { "projects/sample": { bindings: [] } },
  projectsByName: {} as Record<string, { state: number }>,
  roles: [{ name: "roles/example" }],
  workspaceResourceName: () => "workspaces/example",
  loadWorkspacePermissionState: vi.fn(),
  loadProjectIamPolicy: vi.fn(),
  hasWorkspacePermission: vi.fn(),
  hasProjectPermission: vi.fn(),
  fetchDatabases: vi.fn(),
  fetchInstanceList: vi.fn(),
  listIssues: vi.fn(),
};

const mocks = vi.hoisted(() => ({
  searchIssues: vi.fn(),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: {
    searchIssues: mocks.searchIssues,
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => state,
  },
}));

import {
  collectProjectOverview,
  createProjectOverviewTool,
  extractProjectOverviewEvidence,
} from "./projectOverview";

const issue = (values: Partial<Issue>): Issue =>
  ({
    name: "projects/sample/issues/issue-1",
    title: "Add customer table",
    status: IssueStatus.OPEN,
    approvalStatus: ApprovalStatus.PENDING,
    creator: "users/ed@bytebase.com",
    ...values,
  }) as Issue;

describe("projectOverview", () => {
  beforeEach(() => {
    state.roles = [{ name: "roles/example" }];
    state.serverInfo.defaultProject = "projects/default";
    state.projectsByName = {};
    state.listIssues.mockReset();
    state.fetchDatabases.mockReset();
    state.fetchInstanceList.mockReset();
    state.loadWorkspacePermissionState.mockReset().mockResolvedValue(undefined);
    state.loadProjectIamPolicy.mockReset().mockResolvedValue(undefined);
    state.hasWorkspacePermission.mockReset().mockReturnValue(true);
    state.hasProjectPermission.mockReset().mockReturnValue(true);
    state.fetchInstanceList.mockResolvedValue({ instances: [] });
    mocks.searchIssues.mockReset();
  });

  test("discovers an eligible default-project database for an empty project", async () => {
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockImplementation(async ({ parent }) => ({
      nextPageToken: "",
      databases:
        parent === "workspaces/example"
          ? [{ name: "instances/example/databases/unassigned" }]
          : [],
    }));

    const overview = await collectProjectOverview("projects/sample");

    expect(state.fetchDatabases).toHaveBeenCalledWith({
      parent: "workspaces/example",
      pageSize: 1,
      filter: { project: "projects/default" },
      silent: true,
    });
    expect(overview?.setup).toMatchObject({
      instance: "empty",
      transferableDatabase: "present",
      canTransferDatabases: true,
      recommendedStep: "transferDatabases",
    });
  });

  test("uses an existing workspace instance when no transferable database exists", async () => {
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [],
    });
    state.fetchInstanceList.mockImplementation(async ({ parent }) => ({
      instances: parent ? [] : [{ name: "instances/shared" }],
    }));

    const overview = await collectProjectOverview("projects/sample");

    expect(state.fetchInstanceList).toHaveBeenCalledWith({
      parent: "projects/sample",
      pageSize: 1,
      filter: { state: expect.any(Number) },
      silent: true,
    });
    expect(state.fetchInstanceList).toHaveBeenCalledWith({
      parent: undefined,
      pageSize: 1,
      filter: { state: expect.any(Number) },
      silent: true,
    });
    expect(overview?.setup).toMatchObject({
      instance: "present",
      transferableDatabase: "empty",
      recommendedStep: "createDatabase",
    });
  });

  test("does not offer setup actions when known permissions deny them", async () => {
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [],
    });
    state.hasProjectPermission.mockReturnValue(false);
    state.hasWorkspacePermission.mockReturnValue(false);

    const overview = await collectProjectOverview("projects/sample");

    expect(overview?.setup).toMatchObject({
      instance: "unknown",
      transferableDatabase: "empty",
      canConnectInstance: false,
      canTransferDatabases: false,
      canCreateDatabase: false,
    });
    expect(state.fetchInstanceList).not.toHaveBeenCalled();
    expect(state.fetchDatabases).toHaveBeenCalledTimes(1);
  });

  test("does not turn unavailable IAM data into a false permission denial", async () => {
    state.roles = [];
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [],
    });
    state.hasProjectPermission.mockReturnValue(false);
    state.hasWorkspacePermission.mockReturnValue(false);

    const overview = await collectProjectOverview("projects/sample");

    expect(overview?.setup).toMatchObject({
      canConnectInstance: true,
      canTransferDatabases: true,
      canCreateDatabase: true,
    });
  });

  test("preserves unknown availability when setup reads fail", async () => {
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockImplementation(async ({ parent }) => {
      if (parent === "workspaces/example") throw new Error("unavailable");
      return { nextPageToken: "", databases: [] };
    });
    state.fetchInstanceList.mockRejectedValue(new Error("unavailable"));

    const overview = await collectProjectOverview("projects/sample");

    expect(overview?.setup).toMatchObject({
      instance: "unknown",
      transferableDatabase: "unknown",
    });
  });

  test("does not suggest setup for an archived project", async () => {
    state.projectsByName = { "projects/sample": { state: State.DELETED } };
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [],
    });

    const overview = await collectProjectOverview("projects/sample");

    expect(overview?.setup).toBeUndefined();
    expect(state.fetchInstanceList).not.toHaveBeenCalled();
  });

  test("does not discover setup actions from a partial database page", async () => {
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "more",
      databases: [],
    });

    const overview = await collectProjectOverview("projects/sample");

    expect(overview?.setup).toBeUndefined();
    expect(state.fetchDatabases).toHaveBeenCalledTimes(1);
    expect(state.fetchInstanceList).not.toHaveBeenCalled();
  });

  test("does not inspect setup resources for the default project", async () => {
    state.serverInfo.defaultProject = "projects/sample";
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [],
    });

    const overview = await collectProjectOverview("projects/sample");

    expect(overview?.setup).toMatchObject({ isDefault: true });
    expect(overview?.setup?.recommendedStep).toBeUndefined();
    expect(state.fetchDatabases).toHaveBeenCalledTimes(1);
    expect(state.fetchInstanceList).not.toHaveBeenCalled();
  });

  test("does not guess a setup action before the default project is known", async () => {
    state.serverInfo.defaultProject = "";
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [],
    });

    const overview = await collectProjectOverview("projects/sample");

    expect(overview?.setup?.recommendedStep).toBeUndefined();
    expect(state.fetchInstanceList).not.toHaveBeenCalled();
  });

  test("uses silent unhydrated issue searches for every issue facet", async () => {
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [],
    });
    mocks.searchIssues.mockResolvedValue({
      nextPageToken: "",
      issues: [],
    });

    await collectProjectOverview("projects/sample");

    expect(mocks.searchIssues).toHaveBeenCalledTimes(3);
    expect(state.listIssues).not.toHaveBeenCalled();
    for (const [, options] of mocks.searchIssues.mock.calls) {
      expect(options?.contextValues.get(silentContextKey)).toBe(true);
    }
  });

  test("collects bounded project evidence from the chat-bound project", async () => {
    mocks.searchIssues
      .mockResolvedValueOnce({
        nextPageToken: "",
        issues: [issue({ name: "projects/sample/issues/review" })],
      })
      .mockResolvedValueOnce({
        nextPageToken: "",
        issues: [issue({ name: "projects/sample/issues/created" })],
      })
      .mockResolvedValueOnce({
        nextPageToken: "",
        issues: [
          issue({
            name: "projects/sample/issues/database-change",
            title: "Add customer table",
            type: Issue_Type.DATABASE_CHANGE,
          }),
        ],
      });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [
        {
          name: "projects/sample/instances/prod/databases/orders",
        },
      ],
    });

    const evidence = await collectProjectOverview("projects/sample");

    expect(state.fetchDatabases).toHaveBeenCalledWith({
      parent: "projects/sample",
      pageSize: 3,
      silent: true,
    });
    expect(mocks.searchIssues).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        parent: "projects/sample",
        filter:
          'current_approver == "users/ed@bytebase.com" && approval_status == "PENDING" && status in ["OPEN"]',
        orderBy: "update_time desc",
        pageSize: 10,
      }),
      expect.anything()
    );
    expect(mocks.searchIssues).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        parent: "projects/sample",
        filter: 'creator == "users/ed@bytebase.com" && status in ["OPEN"]',
        orderBy: "update_time desc",
        pageSize: 10,
      }),
      expect.anything()
    );
    expect(mocks.searchIssues).toHaveBeenNthCalledWith(
      3,
      expect.objectContaining({
        parent: "projects/sample",
        filter: 'status in ["OPEN"] && type in ["DATABASE_CHANGE"]',
        orderBy: "update_time desc",
        pageSize: 5,
      }),
      expect.anything()
    );
    expect(evidence).toEqual({
      project: "projects/sample",
      evidence: {
        project: "projects/sample",
        references: [
          {
            kind: "issue",
            name: "projects/sample/issues/database-change",
            title: "Add customer table",
          },
          {
            kind: "issue",
            name: "projects/sample/issues/review",
            title: "Add customer table",
          },
          {
            kind: "issue",
            name: "projects/sample/issues/created",
            title: "Add customer table",
          },
          {
            kind: "database",
            name: "projects/sample/instances/prod/databases/orders",
          },
        ],
      },
      personalIssues: {
        status: "available",
        partial: false,
        issues: [
          {
            name: "projects/sample/issues/review",
            title: "Add customer table",
            relation: "review",
            approvalStatus: "PENDING",
          },
          {
            name: "projects/sample/issues/created",
            title: "Add customer table",
            relation: "created",
            approvalStatus: "PENDING",
          },
        ],
      },
      databases: {
        status: "available",
        partial: false,
        databases: [
          {
            name: "projects/sample/instances/prod/databases/orders",
          },
        ],
      },
      openDatabaseChanges: {
        status: "available",
        partial: false,
        issues: [
          {
            name: "projects/sample/issues/database-change",
            title: "Add customer table",
          },
        ],
      },
    });
  });

  test("keeps successful facets when another bounded lookup is incomplete", async () => {
    mocks.searchIssues
      .mockRejectedValueOnce(new Error("approval lookup failed"))
      .mockResolvedValueOnce({
        nextPageToken: "",
        issues: [issue({ name: "projects/sample/issues/created" })],
      })
      .mockResolvedValueOnce({
        nextPageToken: "more-database-changes",
        issues: [
          issue({
            name: "projects/sample/issues/database-change",
            type: Issue_Type.DATABASE_CHANGE,
          }),
        ],
      });
    state.fetchDatabases.mockRejectedValue(new Error("database lookup failed"));

    const evidence = await collectProjectOverview("projects/sample");

    expect(evidence).toEqual({
      project: "projects/sample",
      evidence: {
        project: "projects/sample",
        references: [
          {
            kind: "issue",
            name: "projects/sample/issues/database-change",
            title: "Add customer table",
          },
          {
            kind: "issue",
            name: "projects/sample/issues/created",
            title: "Add customer table",
          },
        ],
      },
      personalIssues: {
        status: "available",
        partial: true,
        issues: [
          {
            name: "projects/sample/issues/created",
            title: "Add customer table",
            relation: "created",
            approvalStatus: "PENDING",
          },
        ],
      },
      databases: { status: "unavailable" },
      openDatabaseChanges: {
        status: "available",
        partial: true,
        issues: [
          {
            name: "projects/sample/issues/database-change",
            title: "Add customer table",
          },
        ],
      },
    });
  });

  test("refuses to inspect a chat that is not bound to a project", async () => {
    await expect(
      collectProjectOverview("workspaces/default")
    ).resolves.toBeNull();
    expect(state.fetchDatabases).not.toHaveBeenCalled();
    expect(mocks.searchIssues).not.toHaveBeenCalled();
  });

  test("returns a project-bound JSON snapshot for the Agent", async () => {
    mocks.searchIssues.mockResolvedValue({ nextPageToken: "", issues: [] });
    state.fetchDatabases.mockResolvedValue({
      nextPageToken: "",
      databases: [],
    });

    await expect(
      createProjectOverviewTool({ projectName: "projects/sample" })()
    ).resolves.toContain('"project":"projects/sample"');
  });

  test("extracts only valid evidence references from an overview result", () => {
    expect(
      extractProjectOverviewEvidence(
        JSON.stringify({
          evidence: {
            project: "projects/sample",
            references: [
              {
                kind: "issue",
                name: "projects/sample/issues/1",
                title: "Add customer table",
              },
              {
                kind: "database",
                name: "projects/sample/instances/prod/databases/orders",
              },
              { kind: "issue", name: 1, title: "Ignored" },
            ],
          },
        })
      )
    ).toEqual({
      project: "projects/sample",
      references: [
        {
          kind: "issue",
          name: "projects/sample/issues/1",
          title: "Add customer table",
        },
        {
          kind: "database",
          name: "projects/sample/instances/prod/databases/orders",
        },
      ],
    });
  });
});
