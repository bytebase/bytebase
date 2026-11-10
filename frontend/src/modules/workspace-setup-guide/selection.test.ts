import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  storageKeyWorkspaceSetupDraft,
  storageKeyWorkspaceSetupFinished,
  storageKeyWorkspaceSetupGuideScenario,
  storageKeyWorkspaceSetupGuideTeamSize,
  storageKeyWorkspaceSetupGuideWorkspaceUsage,
} from "@/utils/storage-keys";
import { isGuideScenarioId } from "./scenarios";
import {
  hasWorkspaceSetupDraft,
  isGuideWorkspaceUsage,
  readGuideTeamSize,
  readGuideWorkspaceUsage,
  readSelectedGuideScenarioId,
  saveGuideTeamSize,
  saveSelectedGuideScenarioId,
} from "./selection";
import {
  readWorkspaceSetupFinished,
  saveWorkspaceSetupFinished,
} from "./setup";

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

  test("accepts only registered scenario ids", () => {
    expect(isGuideScenarioId("query-data")).toBe(true);
    expect(isGuideScenarioId("create-database-change")).toBe(true);
    expect(isGuideScenarioId("mark-sensitive-data")).toBe(true);
    expect(isGuideScenarioId("learn-bytebase-basics")).toBe(false);
    expect(isGuideScenarioId("protect-sensitive-data")).toBe(false);
    expect(isGuideScenarioId("toString")).toBe(false);
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

  test("stores just exploring as a goal of its own", () => {
    expect(saveSelectedGuideScenarioId("just-exploring")).toBe(true);

    expect(readSelectedGuideScenarioId()).toBe("just-exploring");
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

  test.each(["team", "solo"] as const)(
    "reads the %s answer an earlier setup stored",
    (value) => {
      seedWorkspaceUsage(value);

      expect(readGuideWorkspaceUsage()).toBe(value);
    }
  );

  test.each(["2-10", "11-50", "51-200", "200+"] as const)(
    "reads a team of %s as team usage",
    (size) => {
      expect(saveGuideTeamSize(size)).toBe(true);

      expect(readGuideTeamSize()).toBe(size);
      expect(readGuideWorkspaceUsage()).toBe("team");
    }
  );

  test("reads Just me as solo usage, over an earlier stored answer", () => {
    seedWorkspaceUsage("team");
    expect(saveGuideTeamSize("just-me")).toBe(true);

    expect(readGuideWorkspaceUsage()).toBe("solo");
  });

  const teamSizeStorageKey = () =>
    storageKeyWorkspaceSetupGuideTeamSize(
      mocks.state.currentUser.workspace,
      mocks.state.currentUser.email
    );

  test("keeps the team size under its own key, per workspace and user", () => {
    expect(saveGuideTeamSize("11-50")).toBe(true);

    expect(teamSizeStorageKey()).toBe(
      "bb.workspace-setup-guide.team-size.workspaces/ws1.ed@example.com"
    );
    expect(localStorage.getItem(teamSizeStorageKey())).toBe('"11-50"');

    mocks.state.currentUser.workspace = "workspaces/ws2";
    expect(readGuideTeamSize()).toBeUndefined();
  });

  test("ignores a team size it does not know", () => {
    localStorage.setItem(teamSizeStorageKey(), JSON.stringify("1000+"));

    expect(readGuideTeamSize()).toBeUndefined();
    expect(readGuideWorkspaceUsage()).toBeUndefined();
  });
});

describe("workspace setup completion", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  test("scopes completion by workspace without a user suffix", () => {
    expect(storageKeyWorkspaceSetupFinished("workspaces/ws1")).toBe(
      "bb.workspace-setup.finished.workspaces/ws1"
    );
    expect(saveWorkspaceSetupFinished("workspaces/ws1", false)).toBe(true);

    expect(readWorkspaceSetupFinished("workspaces/ws1")).toBe(false);
    expect(readWorkspaceSetupFinished("workspaces/ws2")).toBeUndefined();
  });

  test("distinguishes finished from legacy workspaces", () => {
    expect(readWorkspaceSetupFinished("workspaces/ws1")).toBeUndefined();

    expect(saveWorkspaceSetupFinished("workspaces/ws1", true)).toBe(true);

    expect(readWorkspaceSetupFinished("workspaces/ws1")).toBe(true);
  });

  test("ignores malformed stored completion values", () => {
    localStorage.setItem(
      storageKeyWorkspaceSetupFinished("workspaces/ws1"),
      JSON.stringify("false")
    );

    expect(readWorkspaceSetupFinished("workspaces/ws1")).toBeUndefined();
  });
});

describe("workspace setup draft", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  test("is found only for this workspace and user", () => {
    localStorage.setItem(
      storageKeyWorkspaceSetupDraft("workspaces/ws2", "ed@example.com"),
      "{}"
    );
    localStorage.setItem(
      storageKeyWorkspaceSetupDraft("workspaces/ws1", "ann@example.com"),
      "{}"
    );
    expect(hasWorkspaceSetupDraft()).toBe(false);

    localStorage.setItem(
      storageKeyWorkspaceSetupDraft("workspaces/ws1", "ed@example.com"),
      "{}"
    );

    expect(hasWorkspaceSetupDraft()).toBe(true);
  });
});
