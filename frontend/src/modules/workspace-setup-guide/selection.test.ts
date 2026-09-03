import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  storageKeyWorkspaceSetupGuideScenario,
  storageKeyWorkspaceSetupGuideWorkspaceUsage,
} from "@/utils/storage-keys";
import {
  clearGuideWorkspaceUsage,
  clearSelectedGuideScenarioId,
  isGuideScenarioId,
  isGuideWorkspaceUsage,
  readGuideWorkspaceUsage,
  readSelectedGuideScenarioId,
  saveGuideWorkspaceUsage,
  saveSelectedGuideScenarioId,
} from "./selection";

const mocks = vi.hoisted(() => ({
  state: {
    currentUser: {
      email: "ed@example.com",
      workspace: "workspaces/ws1",
    },
    serverInfo: {
      workspace: "workspaces/ws1",
    },
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => mocks.state,
  },
}));

const storageKey = (workspace = mocks.state.currentUser.workspace) =>
  storageKeyWorkspaceSetupGuideScenario(
    workspace,
    mocks.state.currentUser.email
  );

const seedStoredValue = (value: unknown) => {
  if (value === undefined) {
    localStorage.removeItem(storageKey());
    return;
  }
  localStorage.setItem(storageKey(), JSON.stringify(value));
};

const workspaceUsageStorageKey = (
  workspace = mocks.state.currentUser.workspace
) =>
  storageKeyWorkspaceSetupGuideWorkspaceUsage(
    workspace,
    mocks.state.currentUser.email
  );

const seedWorkspaceUsage = (value: unknown) => {
  if (value === undefined) {
    localStorage.removeItem(workspaceUsageStorageKey());
    return;
  }
  localStorage.setItem(workspaceUsageStorageKey(), JSON.stringify(value));
};

describe("workspace setup guide scenario selection", () => {
  beforeEach(() => {
    localStorage.clear();
    mocks.state.currentUser.workspace = "workspaces/ws1";
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  test("accepts only registered phase-one scenario ids", () => {
    expect(isGuideScenarioId("query-data")).toBe(true);
    expect(isGuideScenarioId("create-database-change")).toBe(true);
    expect(isGuideScenarioId("learn-bytebase-basics")).toBe(false);
    expect(isGuideScenarioId("protect-sensitive-data")).toBe(false);
    expect(isGuideScenarioId(undefined)).toBe(false);
  });

  test.each([undefined, null, "", "unknown", "learn-bytebase-basics", 42])(
    "returns no scenario for %j",
    (value) => {
      seedStoredValue(value);

      expect(readSelectedGuideScenarioId()).toBeUndefined();
    }
  );

  test("falls back to basics when stored JSON is malformed", () => {
    localStorage.setItem(storageKey(), "{");

    expect(readSelectedGuideScenarioId()).toBeUndefined();
  });

  test("stores an explicit valid selection", () => {
    expect(saveSelectedGuideScenarioId("create-database-change")).toBe(true);
    expect(JSON.parse(localStorage.getItem(storageKey()) ?? "null")).toBe(
      "create-database-change"
    );
  });

  test("clears a previous selection", () => {
    expect(saveSelectedGuideScenarioId("query-data")).toBe(true);

    expect(clearSelectedGuideScenarioId()).toBe(true);
    expect(readSelectedGuideScenarioId()).toBeUndefined();
  });

  test("isolates selections by workspace", () => {
    expect(saveSelectedGuideScenarioId("query-data")).toBe(true);

    mocks.state.currentUser.workspace = "workspaces/ws2";

    expect(readSelectedGuideScenarioId()).toBeUndefined();
    expect(localStorage.getItem(storageKey("workspaces/ws1"))).not.toBeNull();
  });

  test("does not throw when localStorage rejects a write", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("quota");
    });

    expect(saveSelectedGuideScenarioId("query-data")).toBe(false);
  });
});

describe("workspace setup guide workspace usage", () => {
  beforeEach(() => {
    localStorage.clear();
    mocks.state.currentUser.workspace = "workspaces/ws1";
  });

  test("accepts only registered workspace usage values", () => {
    expect(isGuideWorkspaceUsage("team")).toBe(true);
    expect(isGuideWorkspaceUsage("solo")).toBe(true);
    expect(isGuideWorkspaceUsage("unknown")).toBe(false);
    expect(isGuideWorkspaceUsage(undefined)).toBe(false);
  });

  test.each([undefined, null, "", "unknown", 42])(
    "returns no workspace usage for %j",
    (value) => {
      seedWorkspaceUsage(value);
      expect(readGuideWorkspaceUsage()).toBeUndefined();
    }
  );

  test.each(["team", "solo"] as const)("stores %s workspace usage", (value) => {
    expect(saveGuideWorkspaceUsage(value)).toBe(true);
    expect(readGuideWorkspaceUsage()).toBe(value);
  });

  test("clears a previous workspace usage", () => {
    expect(saveGuideWorkspaceUsage("team")).toBe(true);
    expect(clearGuideWorkspaceUsage()).toBe(true);
    expect(readGuideWorkspaceUsage()).toBeUndefined();
  });
});
