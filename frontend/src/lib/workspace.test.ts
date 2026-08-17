import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  workspace: "workspaces/exposed-by-auth-info",
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      workspaceResourceName: () => mocks.workspace,
    }),
  },
}));

import { resolveWorkspaceName } from "./workspace";

beforeEach(() => {
  mocks.workspace = "workspaces/exposed-by-auth-info";
  window.history.replaceState(null, "", "/auth");
});

afterEach(() => {
  window.history.replaceState(null, "", "/");
});

describe("resolveWorkspaceName", () => {
  test("uses the workspace exposed by authentication info", () => {
    expect(resolveWorkspaceName()).toBe("workspaces/exposed-by-auth-info");
  });

  test("falls back to the workspace hint from the URL", () => {
    mocks.workspace = "";
    window.history.replaceState(null, "", "/auth?workspace=invited");

    expect(resolveWorkspaceName()).toBe("workspaces/invited");
  });
});
