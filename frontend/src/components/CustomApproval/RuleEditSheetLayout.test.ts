import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const componentDir = dirname(fileURLToPath(import.meta.url));

describe("RuleEditSheet layout", () => {
  test("uses shared form primitives in the approval rule drawer", () => {
    const source = readFileSync(
      join(componentDir, "RuleEditSheet.tsx"),
      "utf8"
    );

    expect(source).toContain("<FormFieldGroup>");
    expect(source).toContain("<FormField");
    expect(source).toContain("FormTitle");
    expect(source).not.toContain("FormLabel");
    expect(source).toContain('id="approval-rule-title"');
    expect(source).toContain('id="approval-rule-description"');
    expect(source).not.toContain('className="flex flex-col gap-y-2"');
    expect(source).not.toContain(
      'className="text-sm font-medium text-control"'
    );
    expect(source).not.toContain('className="textlabel"');
    expect(source).not.toContain('className="textinfolabel text-sm"');
  });
});
