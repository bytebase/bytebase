// @vitest-environment node
import { describe, expect, test } from "vitest";
import { getSkill } from "./index";

describe("query skill", () => {
  test("keeps no-persistence SQL review out of the auto-saving editor", () => {
    const guidance = getSkill({ name: "query" });

    expect(guidance).toContain(
      "SQL Editor automatically creates or updates a Saved Query"
    );
    expect(guidance).toContain(
      "Do not promise that SQL entered there is unsaved or unpersisted"
    );
    expect(guidance).toContain(
      "If the member forbids saving or persistence, do not type SQL into SQL Editor"
    );
    expect(guidance).toContain("return the SQL inline in chat instead");
    expect(guidance).toContain("do not write a Saved Query through APIs");
    expect(guidance).toContain("Do not equate not executed with not saved");
    expect(guidance).toContain("Do not execute review-only SQL");
    expect(guidance).toContain("Execution workflow (only when requested)");
  });
});
