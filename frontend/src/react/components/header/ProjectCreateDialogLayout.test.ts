import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const componentDir = dirname(fileURLToPath(import.meta.url));

describe("ProjectCreateDialog layout", () => {
  test("uses shared form primitives in the create project drawer", () => {
    const source = readFileSync(
      join(componentDir, "ProjectCreateDialog.tsx"),
      "utf8"
    );

    expect(source).toContain("<FormFieldGroup>");
    expect(source).toContain("<FormField>");
    expect(source).toContain("FormTitle");
    expect(source).not.toContain("FormLabel");
    expect(source).toContain('id="create-project-title"');
    expect(source).not.toContain('className="flex flex-col gap-y-6"');
    expect(source).not.toContain("<label");
  });
});
