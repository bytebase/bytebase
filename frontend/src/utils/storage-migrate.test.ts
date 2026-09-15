// @vitest-environment node
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  STORAGE_KEY_AI_DISMISS,
  STORAGE_KEY_BACK_PATH,
  STORAGE_KEY_LANGUAGE,
  STORAGE_KEY_ONBOARDING,
  STORAGE_KEY_SCHEMA_EDITOR_PREVIEW,
  STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
  STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
  STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
  storageKeySqlEditorLastProject,
} from "./storage-keys";
import { migrateStorageKeys, migrateUserStorage } from "./storage-migrate";

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
    expect(localStorage.getItem(STORAGE_KEY_LANGUAGE)).toBe("zh-CN");
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
    localStorage.setItem(STORAGE_KEY_LANGUAGE, "en-US");
    migrateStorageKeys();
    expect(localStorage.getItem(STORAGE_KEY_LANGUAGE)).toBe("en-US");
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
    expect(localStorage.getItem(storageKeySqlEditorLastProject(""))).toBe(
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
        { id: "tab-1", worksheet: "projects/my-proj/worksheets/ws-1" },
        { id: "tab-2", worksheet: "projects/my-proj/worksheets/ws-2" },
      ]);
      localStorage.setItem(
        "bb.sql-editor-tab.projects/my-proj.user@example.com.opening-tab-list",
        tabs
      );
      migrateStorageKeys();
      // The v2 stage renames the legacy `worksheet` link field and its name
      // format in the same pass.
      expect(
        localStorage.getItem(
          "bb.sql-editor.tabs.projects/my-proj.user@example.com"
        )
      ).toBe(
        JSON.stringify([
          { id: "tab-1", savedQuery: "projects/my-proj/savedQueries/ws-1" },
          { id: "tab-2", savedQuery: "projects/my-proj/savedQueries/ws-2" },
        ])
      );
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
        { id: "tab-a", worksheet: "projects/p1/worksheets/ws-a" },
        { id: "tab-b", worksheet: "projects/p1/worksheets/ws-b" },
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
      ).toBe(
        JSON.stringify([
          { id: "tab-a", savedQuery: "projects/p1/savedQueries/ws-a" },
          { id: "tab-b", savedQuery: "projects/p1/savedQueries/ws-b" },
        ])
      );
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
    expect(localStorage.getItem(STORAGE_KEY_LANGUAGE)).toBe("ja-JP");
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT)).toBe(
      "100"
    );
    expect(localStorage.getItem(storageKeySqlEditorLastProject(""))).toBe(
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
    expect(localStorage.length).toBe(2); // only the v1 and v2 markers
  });

  test("preserves unrelated keys", () => {
    localStorage.setItem("some.other.key", "value");
    localStorage.setItem("bb.sql-editor.result-rows-limit", "500");
    migrateStorageKeys();
    expect(localStorage.getItem("some.other.key")).toBe("value");
  });
});

describe("v2: worksheet → saved query rename", () => {
  const MARKER_V2 = "bb.storage-migration-v2";

  test("moves saved-query pref keys, preserving values and scope segments", () => {
    localStorage.setItem(
      "bb.sql-editor.worksheet-filter.projects/p1.a@b.com",
      '{"mine":true}'
    );
    localStorage.setItem(
      "bb.sql-editor.worksheet-tree.ws1.projects/p1.a@b.com",
      '["k1"]'
    );
    localStorage.setItem(
      "bb.sql-editor.worksheet-folder.projects/p1.list.a@b.com",
      '["f1"]'
    );
    migrateStorageKeys();
    expect(
      localStorage.getItem(
        "bb.sql-editor.saved-query-filter.projects/p1.a@b.com"
      )
    ).toBe('{"mine":true}');
    expect(
      localStorage.getItem(
        "bb.sql-editor.saved-query-tree.ws1.projects/p1.a@b.com"
      )
    ).toBe('["k1"]');
    expect(
      localStorage.getItem(
        "bb.sql-editor.saved-query-folder.projects/p1.list.a@b.com"
      )
    ).toBe('["f1"]');
    expect(
      localStorage.getItem("bb.sql-editor.worksheet-filter.projects/p1.a@b.com")
    ).toBeNull();
    expect(localStorage.getItem(MARKER_V2)).toBe("1");
  });

  test("does not overwrite an existing saved-query pref key", () => {
    localStorage.setItem(
      "bb.sql-editor.worksheet-filter.projects/p1.a@b.com",
      "old"
    );
    localStorage.setItem(
      "bb.sql-editor.saved-query-filter.projects/p1.a@b.com",
      "new"
    );
    migrateStorageKeys();
    expect(
      localStorage.getItem(
        "bb.sql-editor.saved-query-filter.projects/p1.a@b.com"
      )
    ).toBe("new");
  });

  test("rewrites persisted tabs already under the current key", () => {
    // Users upgraded past v1 have tabs at the new key with legacy fields.
    localStorage.setItem(MIGRATION_MARKER, "1");
    localStorage.setItem(
      "bb.sql-editor.tabs.projects/p1.a@b.com",
      JSON.stringify([
        {
          id: "t1",
          worksheet: "projects/p1/worksheets/ws-1",
          mode: "WORKSHEET",
        },
        {
          id: "t2",
          savedQuery: "projects/p1/savedQueries/ws-2",
          mode: "ADMIN",
        },
        { id: "t3" },
      ])
    );
    migrateStorageKeys();
    expect(localStorage.getItem("bb.sql-editor.tabs.projects/p1.a@b.com")).toBe(
      JSON.stringify([
        {
          id: "t1",
          savedQuery: "projects/p1/savedQueries/ws-1",
          mode: "SAVED_QUERY",
        },
        {
          id: "t2",
          savedQuery: "projects/p1/savedQueries/ws-2",
          mode: "ADMIN",
        },
        { id: "t3" },
      ])
    );
  });

  test("leaves malformed tab values alone", () => {
    localStorage.setItem("bb.sql-editor.tabs.projects/p1.a@b.com", "not-json");
    migrateStorageKeys();
    expect(localStorage.getItem("bb.sql-editor.tabs.projects/p1.a@b.com")).toBe(
      "not-json"
    );
  });

  test("maps the persisted sidebar tab value", () => {
    localStorage.setItem(
      "bb.sql-editor.sidebar.last-visited-tab.projects/p1",
      '"WORKSHEET"'
    );
    localStorage.setItem(
      "bb.sql-editor.sidebar.last-visited-tab.projects/p2",
      '"SCHEMA"'
    );
    migrateStorageKeys();
    expect(
      localStorage.getItem("bb.sql-editor.sidebar.last-visited-tab.projects/p1")
    ).toBe('"SAVED_QUERY"');
    expect(
      localStorage.getItem("bb.sql-editor.sidebar.last-visited-tab.projects/p2")
    ).toBe('"SCHEMA"');
  });

  test("skips when the v2 marker is set", () => {
    localStorage.setItem(MARKER_V2, "1");
    localStorage.setItem(
      "bb.sql-editor.worksheet-filter.projects/p1.a@b.com",
      "old"
    );
    migrateStorageKeys();
    expect(
      localStorage.getItem("bb.sql-editor.worksheet-filter.projects/p1.a@b.com")
    ).toBe("old");
  });
});

describe("migrateUserStorage", () => {
  test("migrates keys ending with old email to new email", () => {
    localStorage.setItem("bb.recent-visit.old@example.com", '{"visits":[]}');
    localStorage.setItem("bb.quick-access.old@example.com", '["link1"]');

    migrateUserStorage("old@example.com", "new@example.com");

    expect(localStorage.getItem("bb.recent-visit.new@example.com")).toBe(
      '{"visits":[]}'
    );
    expect(localStorage.getItem("bb.quick-access.new@example.com")).toBe(
      '["link1"]'
    );
    expect(localStorage.getItem("bb.recent-visit.old@example.com")).toBeNull();
    expect(localStorage.getItem("bb.quick-access.old@example.com")).toBeNull();
  });

  test("migrates keys with variable middle segments", () => {
    localStorage.setItem(
      "bb.sql-editor.tabs.projects/my-proj.old@example.com",
      '[{"id":"tab1"}]'
    );
    localStorage.setItem(
      "bb.sql-editor.conn-expanded.environments/prod.old@example.com",
      '{"expanded":true}'
    );

    migrateUserStorage("old@example.com", "new@example.com");

    expect(
      localStorage.getItem(
        "bb.sql-editor.tabs.projects/my-proj.new@example.com"
      )
    ).toBe('[{"id":"tab1"}]');
    expect(
      localStorage.getItem(
        "bb.sql-editor.conn-expanded.environments/prod.new@example.com"
      )
    ).toBe('{"expanded":true}');
  });

  test("does not migrate keys not ending with old email", () => {
    localStorage.setItem("bb.language", '"en-US"');
    localStorage.setItem("bb.onboarding", '{"step":1}');
    localStorage.setItem("bb.recent-visit.other@example.com", '{"x":1}');

    migrateUserStorage("old@example.com", "new@example.com");

    expect(localStorage.getItem("bb.language")).toBe('"en-US"');
    expect(localStorage.getItem("bb.onboarding")).toBe('{"step":1}');
    expect(localStorage.getItem("bb.recent-visit.other@example.com")).toBe(
      '{"x":1}'
    );
  });

  test("overwrites existing keys at new location", () => {
    localStorage.setItem("bb.recent-visit.old@example.com", '{"old":true}');
    localStorage.setItem("bb.recent-visit.new@example.com", '{"new":true}');

    migrateUserStorage("old@example.com", "new@example.com");

    expect(localStorage.getItem("bb.recent-visit.new@example.com")).toBe(
      '{"old":true}'
    );
  });

  test("returns early when oldEmail is empty", () => {
    localStorage.setItem("bb.recent-visit.new@example.com", '{"x":1}');

    migrateUserStorage("", "new@example.com");

    expect(localStorage.getItem("bb.recent-visit.new@example.com")).toBe(
      '{"x":1}'
    );
  });

  test("returns early when newEmail is empty", () => {
    localStorage.setItem("bb.recent-visit.old@example.com", '{"x":1}');

    migrateUserStorage("old@example.com", "");

    expect(localStorage.getItem("bb.recent-visit.old@example.com")).toBe(
      '{"x":1}'
    );
  });

  test("returns early when emails are the same", () => {
    localStorage.setItem("bb.recent-visit.same@example.com", '{"x":1}');

    migrateUserStorage("same@example.com", "same@example.com");

    expect(localStorage.getItem("bb.recent-visit.same@example.com")).toBe(
      '{"x":1}'
    );
  });

  test("handles empty localStorage gracefully", () => {
    migrateUserStorage("old@example.com", "new@example.com");

    expect(localStorage.length).toBe(0);
  });

  test("migrates all 17 key families", () => {
    const email = "user@test.com";
    const newEmail = "new@test.com";

    // Set up all key families
    localStorage.setItem(`bb.recent-visit.${email}`, "1");
    localStorage.setItem(`bb.recent-projects.${email}`, "2");
    localStorage.setItem(`bb.quick-access.${email}`, "3");
    localStorage.setItem(`bb.last-activity.${email}`, "4");
    localStorage.setItem(`bb.collapse-state.${email}`, "5");
    localStorage.setItem(`bb.intro-state.${email}`, "6");
    localStorage.setItem(`bb.iam-remind.${email}`, "7");
    localStorage.setItem(`bb.reset-password.${email}`, "8");
    localStorage.setItem(`bb.search./dashboard.${email}`, "9");
    localStorage.setItem(`bb.sql-editor.tabs.projects/p1.${email}`, "10");
    localStorage.setItem(
      `bb.sql-editor.current-tab.projects/p1.${email}`,
      "11"
    );
    localStorage.setItem(`bb.sql-editor.conn-expanded.env/prod.${email}`, "12");
    localStorage.setItem(`bb.sql-editor.conn-expanded-keys.${email}`, "13");
    localStorage.setItem(
      `bb.sql-editor.saved-query-filter.projects/p1.${email}`,
      "14"
    );
    localStorage.setItem(
      `bb.sql-editor.saved-query-tree.projects/p1.${email}`,
      "15"
    );
    localStorage.setItem(
      `bb.sql-editor.saved-query-folder.projects/p1.list.${email}`,
      "16"
    );
    localStorage.setItem(`bb.sql-editor.ai-suggestion.${email}`, "17");

    migrateUserStorage(email, newEmail);

    // Verify all migrated
    expect(localStorage.getItem(`bb.recent-visit.${newEmail}`)).toBe("1");
    expect(localStorage.getItem(`bb.recent-projects.${newEmail}`)).toBe("2");
    expect(localStorage.getItem(`bb.quick-access.${newEmail}`)).toBe("3");
    expect(localStorage.getItem(`bb.last-activity.${newEmail}`)).toBe("4");
    expect(localStorage.getItem(`bb.collapse-state.${newEmail}`)).toBe("5");
    expect(localStorage.getItem(`bb.intro-state.${newEmail}`)).toBe("6");
    expect(localStorage.getItem(`bb.iam-remind.${newEmail}`)).toBe("7");
    expect(localStorage.getItem(`bb.reset-password.${newEmail}`)).toBe("8");
    expect(localStorage.getItem(`bb.search./dashboard.${newEmail}`)).toBe("9");
    expect(
      localStorage.getItem(`bb.sql-editor.tabs.projects/p1.${newEmail}`)
    ).toBe("10");
    expect(
      localStorage.getItem(`bb.sql-editor.current-tab.projects/p1.${newEmail}`)
    ).toBe("11");
    expect(
      localStorage.getItem(`bb.sql-editor.conn-expanded.env/prod.${newEmail}`)
    ).toBe("12");
    expect(
      localStorage.getItem(`bb.sql-editor.conn-expanded-keys.${newEmail}`)
    ).toBe("13");
    expect(
      localStorage.getItem(
        `bb.sql-editor.saved-query-filter.projects/p1.${newEmail}`
      )
    ).toBe("14");
    expect(
      localStorage.getItem(
        `bb.sql-editor.saved-query-tree.projects/p1.${newEmail}`
      )
    ).toBe("15");
    expect(
      localStorage.getItem(
        `bb.sql-editor.saved-query-folder.projects/p1.list.${newEmail}`
      )
    ).toBe("16");
    expect(
      localStorage.getItem(`bb.sql-editor.ai-suggestion.${newEmail}`)
    ).toBe("17");

    // Verify old keys removed
    expect(localStorage.getItem(`bb.recent-visit.${email}`)).toBeNull();
    expect(
      localStorage.getItem(`bb.sql-editor.tabs.projects/p1.${email}`)
    ).toBeNull();
  });
});
