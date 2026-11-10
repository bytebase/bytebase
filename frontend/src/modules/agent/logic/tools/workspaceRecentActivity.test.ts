// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({ getRecentActivity: vi.fn() }));

vi.mock("./recentActivity", () => ({
  getRecentActivity: mocks.getRecentActivity,
}));

import { createWorkspaceRecentActivityTool } from "./workspaceRecentActivity";

describe("createWorkspaceRecentActivityTool", () => {
  beforeEach(() => {
    mocks.getRecentActivity.mockReset();
  });

  test("uses the workspace-wide project audit scope", async () => {
    mocks.getRecentActivity.mockResolvedValue(
      JSON.stringify({ events: [], partial: false })
    );

    await expect(
      createWorkspaceRecentActivityTool({ workspaceName: "workspaces/acme" })()
    ).resolves.toBe(
      JSON.stringify({
        workspace: "workspaces/acme",
        events: [],
        partial: false,
      })
    );
    expect(mocks.getRecentActivity).toHaveBeenCalledWith("projects/-");
  });

  test("rejects an unbound workspace without searching audit logs", async () => {
    await expect(createWorkspaceRecentActivityTool({})()).resolves.toBe(
      JSON.stringify({ error: "This starter is not bound to a workspace." })
    );
    expect(mocks.getRecentActivity).not.toHaveBeenCalled();
  });
});
