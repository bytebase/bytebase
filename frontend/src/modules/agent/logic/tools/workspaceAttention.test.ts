// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import {
  IssueSchema,
  SearchIssuesRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";

const mocks = vi.hoisted(() => ({ searchIssues: vi.fn() }));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { searchIssues: mocks.searchIssues },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ currentUser: { name: "users/ed@bytebase.com" } }),
  },
}));

import { createWorkspaceAttentionTool } from "./workspaceAttention";

describe("createWorkspaceAttentionTool", () => {
  beforeEach(() => {
    mocks.searchIssues.mockReset();
  });

  test("lists the current user's pending approvals and open Issues", async () => {
    mocks.searchIssues
      .mockResolvedValueOnce({
        issues: [
          create(IssueSchema, {
            name: "projects/orders/issues/42",
            title: "Add customer table",
          }),
        ],
        nextPageToken: "next-page",
      })
      .mockResolvedValueOnce({
        issues: [
          create(IssueSchema, {
            name: "projects/orders/issues/43",
            title: "Backfill customer data",
          }),
          create(IssueSchema, {
            name: "projects/orders/issues/42",
            title: "Add customer table",
          }),
        ],
        nextPageToken: "",
      });

    await expect(
      createWorkspaceAttentionTool({ workspaceName: "workspaces/acme" })()
    ).resolves.toBe(
      JSON.stringify({
        issues: [
          {
            name: "projects/orders/issues/42",
            title: "Add customer table",
            relation: "review",
          },
          {
            name: "projects/orders/issues/43",
            title: "Backfill customer data",
            relation: "created",
          },
        ],
        partial: true,
      })
    );

    expect(mocks.searchIssues).toHaveBeenNthCalledWith(
      1,
      create(SearchIssuesRequestSchema, {
        parent: "projects/-",
        filter:
          'current_approver == "users/ed@bytebase.com" && approval_status == "PENDING" && status in ["OPEN"]',
        orderBy: "update_time desc",
        pageSize: 5,
      }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(mocks.searchIssues).toHaveBeenNthCalledWith(
      2,
      create(SearchIssuesRequestSchema, {
        parent: "projects/-",
        filter: 'creator == "users/ed@bytebase.com" && status in ["OPEN"]',
        orderBy: "update_time desc",
        pageSize: 5,
      }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(
      mocks.searchIssues.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
    expect(
      mocks.searchIssues.mock.calls[1][1].contextValues.get(silentContextKey)
    ).toBe(true);
  });

  test("refuses an unbound workspace without searching Issues", async () => {
    await expect(createWorkspaceAttentionTool({})()).resolves.toBe(
      JSON.stringify({ error: "This starter is not bound to a workspace." })
    );
    expect(mocks.searchIssues).not.toHaveBeenCalled();
  });
});
