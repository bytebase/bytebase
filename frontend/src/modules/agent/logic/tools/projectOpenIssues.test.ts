// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";

const state = {
  currentUser: { name: "users/ed@bytebase.com" },
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

import { createProjectOpenIssuesTool } from "./projectOpenIssues";

const issue = (values: Partial<Issue>): Issue =>
  ({
    name: "projects/sample/issues/issue-1",
    title: "Add customer table",
    status: IssueStatus.OPEN,
    approvalStatus: ApprovalStatus.PENDING,
    creator: "users/ed@bytebase.com",
    ...values,
  }) as Issue;

describe("projectOpenIssues", () => {
  beforeEach(() => {
    state.listIssues.mockReset();
    mocks.searchIssues.mockReset();
  });

  test("uses silent unhydrated issue searches for best-effort discovery", async () => {
    mocks.searchIssues.mockResolvedValue({
      nextPageToken: "",
      issues: [],
    });

    await createProjectOpenIssuesTool({ projectName: "projects/sample" })();

    expect(mocks.searchIssues).toHaveBeenCalledTimes(2);
    expect(state.listIssues).not.toHaveBeenCalled();
    for (const [, options] of mocks.searchIssues.mock.calls) {
      expect(options?.contextValues.get(silentContextKey)).toBe(true);
    }
  });

  test("lists the current member's open Project issues without navigating", async () => {
    mocks.searchIssues
      .mockResolvedValueOnce({
        nextPageToken: "",
        issues: [issue({ name: "projects/sample/issues/review" })],
      })
      .mockResolvedValueOnce({
        nextPageToken: "",
        issues: [
          issue({ name: "projects/sample/issues/created" }),
          issue({ name: "projects/sample/issues/review" }),
        ],
      });

    const result = JSON.parse(
      await createProjectOpenIssuesTool({ projectName: "projects/sample" })()
    );

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
    expect(result).toEqual({
      project: "projects/sample",
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
      partial: false,
    });
  });

  test("refuses to query a chat that is not bound to a Project", async () => {
    const result = JSON.parse(
      await createProjectOpenIssuesTool({ projectName: "workspaces/default" })()
    );

    expect(result).toEqual({
      error: "This starter is not bound to a Project.",
    });
    expect(mocks.searchIssues).not.toHaveBeenCalled();
  });

  test("marks the result partial when either Issue query has another page", async () => {
    mocks.searchIssues
      .mockResolvedValueOnce({
        nextPageToken: "next-review-page",
        issues: [issue({ name: "projects/sample/issues/review" })],
      })
      .mockResolvedValueOnce({
        nextPageToken: "",
        issues: [issue({ name: "projects/sample/issues/created" })],
      });

    const result = JSON.parse(
      await createProjectOpenIssuesTool({ projectName: "projects/sample" })()
    );

    expect(result.partial).toBe(true);
  });
});
