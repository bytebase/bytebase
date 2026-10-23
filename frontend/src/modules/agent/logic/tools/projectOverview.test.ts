// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { type Issue, Issue_Type } from "@/types/proto-es/v1/issue_service_pb";

const state = {
  currentUser: { name: "users/ed@bytebase.com" },
  fetchDatabases: vi.fn(),
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
    state.listIssues.mockReset();
    state.fetchDatabases.mockReset();
    mocks.searchIssues.mockReset();
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
