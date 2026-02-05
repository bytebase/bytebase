import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { migrateUserStorage } from "./storage-migrate";

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
      `bb.sql-editor.worksheet-filter.projects/p1.${email}`,
      "14"
    );
    localStorage.setItem(
      `bb.sql-editor.worksheet-tree.projects/p1.${email}`,
      "15"
    );
    localStorage.setItem(
      `bb.sql-editor.worksheet-folder.projects/p1.list.${email}`,
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
        `bb.sql-editor.worksheet-filter.projects/p1.${newEmail}`
      )
    ).toBe("14");
    expect(
      localStorage.getItem(
        `bb.sql-editor.worksheet-tree.projects/p1.${newEmail}`
      )
    ).toBe("15");
    expect(
      localStorage.getItem(
        `bb.sql-editor.worksheet-folder.projects/p1.list.${newEmail}`
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
