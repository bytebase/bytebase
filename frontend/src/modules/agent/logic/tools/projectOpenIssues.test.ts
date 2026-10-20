// @vitest-environment node
import { describe, expect, test, vi } from "vitest";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";

const state = {
  currentUser: { name: "users/ed@bytebase.com" },
  listIssues: vi.fn(),
};

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
  test("lists the current member's open Project issues without navigating", async () => {
    state.listIssues.mockReset();
    state.listIssues
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

    expect(state.listIssues).toHaveBeenNthCalledWith(1, {
      find: {
        project: "projects/sample",
        query: "",
        currentApprover: "users/ed@bytebase.com",
        statusList: [IssueStatus.OPEN],
        approvalStatus: ApprovalStatus.PENDING,
        orderBy: "update_time desc",
      },
      pageSize: 10,
    });
    expect(state.listIssues).toHaveBeenNthCalledWith(2, {
      find: {
        project: "projects/sample",
        query: "",
        creator: "users/ed@bytebase.com",
        statusList: [IssueStatus.OPEN],
        orderBy: "update_time desc",
      },
      pageSize: 10,
    });
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
    state.listIssues.mockReset();

    const result = JSON.parse(
      await createProjectOpenIssuesTool({ projectName: "workspaces/default" })()
    );

    expect(result).toEqual({
      error: "This starter is not bound to a Project.",
    });
    expect(state.listIssues).not.toHaveBeenCalled();
  });

  test("marks the result partial when either Issue query has another page", async () => {
    state.listIssues.mockReset();
    state.listIssues
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
