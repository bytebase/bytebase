import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const pageSource = readFileSync(
  join(import.meta.dirname, "ProjectSettingsPage.tsx"),
  "utf8"
);

describe("ProjectSettingsPage layout", () => {
  test("uses shared form primitives for settings sections and fields", () => {
    expect(pageSource).toContain("@/components/ui/form");
    expect(pageSource).toContain("FormSection");
    expect(pageSource).toContain("FormFieldGroup");
    expect(pageSource).toContain("<FormField");
    expect(pageSource).toContain("FormControlRow");
    expect(pageSource).toContain('className="divide-y divide-block-border"');
    expect(pageSource).not.toContain('className="py-6 lg:flex"');
    expect(pageSource).not.toContain('className="pb-6 lg:flex"');
    expect(pageSource).not.toContain('className="flex-1 mt-4 lg:px-4 lg:mt-0"');
  });
});
