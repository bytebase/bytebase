import { describe, expect, test } from "vitest";
import {
  getGhostConfig,
  isGhostEnabled,
  updateGhostConfig,
} from "./directiveUtils";

// The backend matches the `-- gh-ost = {...}` directive with a multiline search
// (backend/component/ghost/directive.go), so a directive below the SQL is valid.
// These tests lock the frontend to the same "anywhere" semantics so the Configure
// editor reads and rewrites such directives instead of dropping their flags.
describe("getGhostConfig — directive position", () => {
  test("reads a leading directive", () => {
    const statement = `-- gh-ost = {"chunk-size":"500"}\nALTER TABLE t ADD COLUMN c INT;`;
    expect(getGhostConfig(statement)).toEqual({ "chunk-size": "500" });
  });

  test("reads a directive that follows the SQL", () => {
    const statement = `ALTER TABLE t ADD COLUMN c INT;\n-- gh-ost = {"max-load":"Threads_running=50"}`;
    expect(getGhostConfig(statement)).toEqual({
      "max-load": "Threads_running=50",
    });
  });

  test("detects a non-leading directive as enabled", () => {
    expect(isGhostEnabled("ALTER TABLE t;\n-- gh-ost = {}")).toBe(true);
  });
});

describe("updateGhostConfig — non-leading directive", () => {
  test("rewrites a single leading directive without dropping existing flags", () => {
    const statement = `ALTER TABLE t ADD COLUMN c INT;\n-- gh-ost = {"max-load":"Threads_running=50"}`;
    const updated = updateGhostConfig(statement, {
      "max-load": "Threads_running=50",
      "chunk-size": "500",
    });

    // Exactly one directive — the below-the-SQL one is not left as a duplicate.
    expect(updated.match(/--\s*gh-ost\s*=/g)).toHaveLength(1);
    // It carries both the pre-existing and the newly-edited flag.
    expect(getGhostConfig(updated)).toEqual({
      "max-load": "Threads_running=50",
      "chunk-size": "500",
    });
    expect(updated).toContain("ALTER TABLE t ADD COLUMN c INT;");
  });

  test("removing the directive strips it wherever it sits", () => {
    const statement = `ALTER TABLE t ADD COLUMN c INT;\n-- gh-ost = {"max-load":"x"}`;
    const updated = updateGhostConfig(statement, undefined);

    expect(isGhostEnabled(updated)).toBe(false);
    expect(updated).not.toMatch(/--\s*gh-ost/);
    expect(updated).toContain("ALTER TABLE t ADD COLUMN c INT;");
  });
});
