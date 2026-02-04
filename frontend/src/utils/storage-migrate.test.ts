import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  STORAGE_KEY_AI_DISMISS,
  STORAGE_KEY_BACK_PATH,
  STORAGE_KEY_LANGUAGE,
  STORAGE_KEY_ONBOARDING,
  STORAGE_KEY_SCHEMA_EDITOR_PREVIEW,
  STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
  STORAGE_KEY_SQL_EDITOR_LAST_PROJECT,
  STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
  STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
} from "./storage-keys";
import { migrateStorageKeys } from "./storage-migrate";

const MIGRATION_MARKER = "bb.storage-migration-v1";

function createMockStorage(): Storage {
  let store: Record<string, string> = {};
  return {
    get length() {
      return Object.keys(store).length;
    },
    key(index: number) {
      return Object.keys(store)[index] ?? null;
    },
    getItem(key: string) {
      return store[key] ?? null;
    },
    setItem(key: string, value: string) {
      store[key] = String(value);
    },
    removeItem(key: string) {
      delete store[key];
    },
    clear() {
      store = {};
    },
  };
}

let mockStorage: Storage;

beforeEach(() => {
  mockStorage = createMockStorage();
  vi.stubGlobal("localStorage", mockStorage);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("migrateStorageKeys", () => {
  test("sets migration marker after running", () => {
    migrateStorageKeys();
    expect(localStorage.getItem(MIGRATION_MARKER)).toBe("1");
  });

  test("skips if migration marker already set", () => {
    localStorage.setItem("ui.backPath", "/old");
    localStorage.setItem(MIGRATION_MARKER, "1");

    migrateStorageKeys();

    expect(localStorage.getItem("ui.backPath")).toBe("/old");
    expect(localStorage.getItem(STORAGE_KEY_BACK_PATH)).toBeNull();
  });
});

describe("static key renames", () => {
  test("migrates ui.backPath to bb.back-path", () => {
    localStorage.setItem("ui.backPath", "/some/path");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_BACK_PATH)).toBe("/some/path");
    expect(localStorage.getItem("ui.backPath")).toBeNull();
  });

  test("migrates bb.onboarding-state to bb.onboarding", () => {
    const state = JSON.stringify({ isOnboarding: true, consumed: ["step1"] });
    localStorage.setItem("bb.onboarding-state", state);
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_ONBOARDING)).toBe(state);
    expect(localStorage.getItem("bb.onboarding-state")).toBeNull();
  });

  test("migrates bb.schema-editor.preview.expanded", () => {
    localStorage.setItem("bb.schema-editor.preview.expanded", "true");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_SCHEMA_EDITOR_PREVIEW)).toBe(
      "true"
    );
    expect(
      localStorage.getItem("bb.schema-editor.preview.expanded")
    ).toBeNull();
  });

  test("migrates bb.plugin.open-ai.dismiss-placeholder", () => {
    localStorage.setItem("bb.plugin.open-ai.dismiss-placeholder", "true");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_AI_DISMISS)).toBe("true");
  });

  test("migrates bb.plugin.editor.ai-panel-size", () => {
    localStorage.setItem("bb.plugin.editor.ai-panel-size", "0.5");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE)).toBe(
      "0.5"
    );
  });

  test("does not overwrite existing new key", () => {
    localStorage.setItem("ui.backPath", "/old");
    localStorage.setItem(STORAGE_KEY_BACK_PATH, "/new");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_BACK_PATH)).toBe("/new");
    expect(localStorage.getItem("ui.backPath")).toBeNull();
  });
});

describe("language migration", () => {
  test("migrates bytebase_options nested object to flat string", () => {
    localStorage.setItem(
      "bytebase_options",
      JSON.stringify({ appearance: { language: "zh-CN" } })
    );
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_LANGUAGE)).toBe('"zh-CN"');
    expect(localStorage.getItem("bytebase_options")).toBeNull();
  });

  test("skips language migration if language is empty", () => {
    localStorage.setItem(
      "bytebase_options",
      JSON.stringify({ appearance: { language: "" } })
    );
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_LANGUAGE)).toBeNull();
    expect(localStorage.getItem("bytebase_options")).toBeNull();
  });

  test("removes bytebase_options even if new key exists", () => {
    localStorage.setItem(
      "bytebase_options",
      JSON.stringify({ appearance: { language: "zh-CN" } })
    );
    localStorage.setItem(STORAGE_KEY_LANGUAGE, '"en-US"');
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_LANGUAGE)).toBe('"en-US"');
    expect(localStorage.getItem("bytebase_options")).toBeNull();
  });

  test("handles malformed bytebase_options gracefully", () => {
    localStorage.setItem("bytebase_options", "not-json");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_LANGUAGE)).toBeNull();
    expect(localStorage.getItem("bytebase_options")).toBeNull();
  });
});

describe("SQL editor migrations", () => {
  test("migrates bb.sql-editor.result-rows-limit", () => {
    localStorage.setItem("bb.sql-editor.result-rows-limit", "500");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT)).toBe(
      "500"
    );
    expect(localStorage.getItem("bb.sql-editor.result-rows-limit")).toBeNull();
  });

  test("migrates bb.sql-editor.redis-command-node", () => {
    localStorage.setItem("bb.sql-editor.redis-command-node", "1");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_REDIS_NODE)).toBe("1");
    expect(localStorage.getItem("bb.sql-editor.redis-command-node")).toBeNull();
  });

  test("migrates bb.sql-editor.last-viewed-project", () => {
    localStorage.setItem(
      "bb.sql-editor.last-viewed-project",
      '"projects/my-project"'
    );
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_LAST_PROJECT)).toBe(
      '"projects/my-project"'
    );
    expect(
      localStorage.getItem("bb.sql-editor.last-viewed-project")
    ).toBeNull();
  });

  test("does not overwrite existing sql editor keys", () => {
    localStorage.setItem("bb.sql-editor.result-rows-limit", "500");
    localStorage.setItem(STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT, "2000");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT)).toBe(
      "2000"
    );
  });

  describe("connection pane expanded state", () => {
    test("migrates expanded_{env}.{email} keys", () => {
      const state = JSON.stringify({
        initialized: true,
        expandedKeys: ["env-1/db-1"],
      });
      localStorage.setItem(
        "bb.sql-editor.connection-pane.expanded_environments/prod.user@example.com",
        state
      );
      migrateStorageKeys();
      expect(
        localStorage.getItem(
          "bb.sql-editor.conn-expanded.environments/prod.user@example.com"
        )
      ).toBe(state);
      expect(
        localStorage.getItem(
          "bb.sql-editor.connection-pane.expanded_environments/prod.user@example.com"
        )
      ).toBeNull();
    });

    test("migrates multiple environment expanded keys", () => {
      localStorage.setItem(
        "bb.sql-editor.connection-pane.expanded_environments/prod.a@b.com",
        "prod-state"
      );
      localStorage.setItem(
        "bb.sql-editor.connection-pane.expanded_environments/dev.a@b.com",
        "dev-state"
      );
      migrateStorageKeys();
      expect(
        localStorage.getItem(
          "bb.sql-editor.conn-expanded.environments/prod.a@b.com"
        )
      ).toBe("prod-state");
      expect(
        localStorage.getItem(
          "bb.sql-editor.conn-expanded.environments/dev.a@b.com"
        )
      ).toBe("dev-state");
    });

    test("does not overwrite existing conn-expanded key when new key exists", () => {
      localStorage.setItem(
        "bb.sql-editor.connection-pane.expanded_environments/prod.a@b.com",
        "old"
      );
      localStorage.setItem(
        "bb.sql-editor.conn-expanded.environments/prod.a@b.com",
        "new"
      );
      migrateStorageKeys();
      expect(
        localStorage.getItem(
          "bb.sql-editor.conn-expanded.environments/prod.a@b.com"
        )
      ).toBe("new");
    });
  });

  describe("tab keys (opening-tab-list and current-tab-id)", () => {
    test("migrates opening-tab-list to bb.sql-editor.tabs.*", () => {
      const tabs = JSON.stringify([
        { id: "tab-1", worksheet: "worksheets/ws-1" },
        { id: "tab-2", worksheet: "worksheets/ws-2" },
      ]);
      localStorage.setItem(
        "bb.sql-editor-tab.projects/my-proj.user@example.com.opening-tab-list",
        tabs
      );
      migrateStorageKeys();
      expect(
        localStorage.getItem(
          "bb.sql-editor.tabs.projects/my-proj.user@example.com"
        )
      ).toBe(tabs);
      expect(
        localStorage.getItem(
          "bb.sql-editor-tab.projects/my-proj.user@example.com.opening-tab-list"
        )
      ).toBeNull();
    });

    test("migrates current-tab-id to bb.sql-editor.current-tab.*", () => {
      localStorage.setItem(
        "bb.sql-editor-tab.projects/my-proj.user@example.com.current-tab-id",
        '"tab-1"'
      );
      migrateStorageKeys();
      expect(
        localStorage.getItem(
          "bb.sql-editor.current-tab.projects/my-proj.user@example.com"
        )
      ).toBe('"tab-1"');
      expect(
        localStorage.getItem(
          "bb.sql-editor-tab.projects/my-proj.user@example.com.current-tab-id"
        )
      ).toBeNull();
    });

    test("migrates both tab list and current tab together", () => {
      const tabs = JSON.stringify([
        { id: "tab-a", worksheet: "worksheets/ws-a" },
        { id: "tab-b", worksheet: "worksheets/ws-b" },
      ]);
      localStorage.setItem(
        "bb.sql-editor-tab.projects/p1.dev@bb.com.opening-tab-list",
        tabs
      );
      localStorage.setItem(
        "bb.sql-editor-tab.projects/p1.dev@bb.com.current-tab-id",
        '"tab-b"'
      );
      migrateStorageKeys();
      expect(
        localStorage.getItem("bb.sql-editor.tabs.projects/p1.dev@bb.com")
      ).toBe(tabs);
      expect(
        localStorage.getItem("bb.sql-editor.current-tab.projects/p1.dev@bb.com")
      ).toBe('"tab-b"');
    });

    test("migrates tabs for multiple projects", () => {
      localStorage.setItem(
        "bb.sql-editor-tab.projects/p1.a@b.com.opening-tab-list",
        '[{"id":"t1"}]'
      );
      localStorage.setItem(
        "bb.sql-editor-tab.projects/p2.a@b.com.opening-tab-list",
        '[{"id":"t2"}]'
      );
      migrateStorageKeys();
      expect(
        localStorage.getItem("bb.sql-editor.tabs.projects/p1.a@b.com")
      ).toBe('[{"id":"t1"}]');
      expect(
        localStorage.getItem("bb.sql-editor.tabs.projects/p2.a@b.com")
      ).toBe('[{"id":"t2"}]');
    });

    test("does not overwrite existing new tab key", () => {
      localStorage.setItem(
        "bb.sql-editor-tab.projects/p1.a@b.com.opening-tab-list",
        '[{"id":"old"}]'
      );
      localStorage.setItem(
        "bb.sql-editor.tabs.projects/p1.a@b.com",
        '[{"id":"new"}]'
      );
      migrateStorageKeys();
      expect(
        localStorage.getItem("bb.sql-editor.tabs.projects/p1.a@b.com")
      ).toBe('[{"id":"new"}]');
    });

    test("preserves all tabs during migration (no data loss)", () => {
      const twoTabs = [
        {
          id: "tab-1",
          worksheet: "worksheets/ws-1",
          mode: "READONLY",
          viewState: {},
        },
        {
          id: "tab-2",
          worksheet: "worksheets/ws-2",
          mode: "READONLY",
          viewState: {},
        },
      ];
      localStorage.setItem(
        "bb.sql-editor-tab.projects/default.user@test.com.opening-tab-list",
        JSON.stringify(twoTabs)
      );
      localStorage.setItem(
        "bb.sql-editor-tab.projects/default.user@test.com.current-tab-id",
        '"tab-2"'
      );

      migrateStorageKeys();

      const migrated = JSON.parse(
        localStorage.getItem(
          "bb.sql-editor.tabs.projects/default.user@test.com"
        )!
      );
      expect(migrated).toHaveLength(2);
      expect(migrated[0].id).toBe("tab-1");
      expect(migrated[1].id).toBe("tab-2");
      expect(
        localStorage.getItem(
          "bb.sql-editor.current-tab.projects/default.user@test.com"
        )
      ).toBe('"tab-2"');
    });
  });
});

describe("prefix renames", () => {
  test("migrates bb.plugin.open-ai.suggestions.* keys", () => {
    localStorage.setItem(
      "bb.plugin.open-ai.suggestions.abc123",
      '["SELECT 1"]'
    );
    localStorage.setItem(
      "bb.plugin.open-ai.suggestions.def456",
      '["SELECT 2"]'
    );
    migrateStorageKeys();
    expect(localStorage.getItem("bb.ai.suggestions.abc123")).toBe(
      '["SELECT 1"]'
    );
    expect(localStorage.getItem("bb.ai.suggestions.def456")).toBe(
      '["SELECT 2"]'
    );
    expect(
      localStorage.getItem("bb.plugin.open-ai.suggestions.abc123")
    ).toBeNull();
    expect(
      localStorage.getItem("bb.plugin.open-ai.suggestions.def456")
    ).toBeNull();
  });

  test("migrates bb.context-menu-button.* keys", () => {
    localStorage.setItem("bb.context-menu-button.task-transition", '"approve"');
    migrateStorageKeys();
    expect(localStorage.getItem("bb.context-menu.task-transition")).toBe(
      '"approve"'
    );
    expect(
      localStorage.getItem("bb.context-menu-button.task-transition")
    ).toBeNull();
  });
});

describe("UI scope key renames", () => {
  test("migrates ui.list.collapse.{email}", () => {
    const state = JSON.stringify({ section1: true, section2: false });
    localStorage.setItem("ui.list.collapse.user@example.com", state);
    migrateStorageKeys();
    expect(localStorage.getItem("bb.collapse-state.user@example.com")).toBe(
      state
    );
    expect(
      localStorage.getItem("ui.list.collapse.user@example.com")
    ).toBeNull();
  });

  test("migrates ui.intro.{email}", () => {
    const state = JSON.stringify({ tip1: true });
    localStorage.setItem("ui.intro.user@example.com", state);
    migrateStorageKeys();
    expect(localStorage.getItem("bb.intro-state.user@example.com")).toBe(state);
    expect(localStorage.getItem("ui.intro.user@example.com")).toBeNull();
  });

  test("migrates {email}.require_reset_password", () => {
    localStorage.setItem("user@example.com.require_reset_password", "true");
    migrateStorageKeys();
    expect(localStorage.getItem("bb.reset-password.user@example.com")).toBe(
      "true"
    );
    expect(
      localStorage.getItem("user@example.com.require_reset_password")
    ).toBeNull();
  });
});

describe("full migration scenario", () => {
  test("migrates all key types in a single run", () => {
    // Static
    localStorage.setItem("ui.backPath", "/dashboard");
    // Language
    localStorage.setItem(
      "bytebase_options",
      JSON.stringify({ appearance: { language: "ja-JP" } })
    );
    // SQL editor
    localStorage.setItem("bb.sql-editor.result-rows-limit", "100");
    localStorage.setItem("bb.sql-editor.last-viewed-project", '"projects/p1"');
    // Connection expanded
    localStorage.setItem(
      "bb.sql-editor.connection-pane.expanded_environments/staging.dev@bb.com",
      '{"initialized":true,"expandedKeys":["k1"]}'
    );
    // Tab keys
    localStorage.setItem(
      "bb.sql-editor-tab.projects/p1.dev@bb.com.opening-tab-list",
      '[{"id":"t1"},{"id":"t2"}]'
    );
    localStorage.setItem(
      "bb.sql-editor-tab.projects/p1.dev@bb.com.current-tab-id",
      '"t2"'
    );
    // Prefix
    localStorage.setItem("bb.plugin.open-ai.suggestions.hash1", '["s1"]');
    // UI scope
    localStorage.setItem("ui.intro.dev@bb.com", '{"done":true}');

    migrateStorageKeys();

    // Verify all migrated
    expect(localStorage.getItem(STORAGE_KEY_BACK_PATH)).toBe("/dashboard");
    expect(localStorage.getItem(STORAGE_KEY_LANGUAGE)).toBe('"ja-JP"');
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT)).toBe(
      "100"
    );
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_LAST_PROJECT)).toBe(
      '"projects/p1"'
    );
    expect(
      localStorage.getItem(
        "bb.sql-editor.conn-expanded.environments/staging.dev@bb.com"
      )
    ).toBe('{"initialized":true,"expandedKeys":["k1"]}');
    expect(localStorage.getItem("bb.ai.suggestions.hash1")).toBe('["s1"]');
    expect(localStorage.getItem("bb.intro-state.dev@bb.com")).toBe(
      '{"done":true}'
    );
    // Tab keys
    const tabs = JSON.parse(
      localStorage.getItem("bb.sql-editor.tabs.projects/p1.dev@bb.com")!
    );
    expect(tabs).toHaveLength(2);
    expect(
      localStorage.getItem("bb.sql-editor.current-tab.projects/p1.dev@bb.com")
    ).toBe('"t2"');

    // Verify old keys removed
    expect(localStorage.getItem("ui.backPath")).toBeNull();
    expect(localStorage.getItem("bytebase_options")).toBeNull();
    expect(localStorage.getItem("bb.sql-editor.result-rows-limit")).toBeNull();
    expect(
      localStorage.getItem("bb.sql-editor.last-viewed-project")
    ).toBeNull();
    expect(
      localStorage.getItem(
        "bb.sql-editor.connection-pane.expanded_environments/staging.dev@bb.com"
      )
    ).toBeNull();
    expect(
      localStorage.getItem("bb.plugin.open-ai.suggestions.hash1")
    ).toBeNull();
    expect(localStorage.getItem("ui.intro.dev@bb.com")).toBeNull();
    expect(
      localStorage.getItem(
        "bb.sql-editor-tab.projects/p1.dev@bb.com.opening-tab-list"
      )
    ).toBeNull();
    expect(
      localStorage.getItem(
        "bb.sql-editor-tab.projects/p1.dev@bb.com.current-tab-id"
      )
    ).toBeNull();

    // Marker set
    expect(localStorage.getItem(MIGRATION_MARKER)).toBe("1");
  });

  test("handles empty localStorage gracefully", () => {
    migrateStorageKeys();
    expect(localStorage.getItem(MIGRATION_MARKER)).toBe("1");
    expect(localStorage.length).toBe(1); // only the marker
  });

  test("preserves unrelated keys", () => {
    localStorage.setItem("some.other.key", "value");
    localStorage.setItem("bb.sql-editor.result-rows-limit", "500");
    migrateStorageKeys();
    expect(localStorage.getItem("some.other.key")).toBe("value");
  });
});
