import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const sectionDir = dirname(fileURLToPath(import.meta.url));

describe("RolesPage drawer layout", () => {
  test("uses shared form primitives in the role sheet", () => {
    const source = readFileSync(join(sectionDir, "RolesPage.tsx"), "utf8");
    const roleSheetSource = source.slice(
      source.indexOf("function RoleSheet"),
      source.indexOf(
        "// ============================================================\n// RolesPage"
      )
    );

    expect(roleSheetSource).toContain("<FormFieldGroup>");
    expect(roleSheetSource).toContain("<FormField");
    expect(roleSheetSource).toContain("FormTitle");
    expect(roleSheetSource).toContain('id="role-sheet-title"');
    expect(roleSheetSource).toContain('id="role-sheet-description"');
    expect(roleSheetSource).not.toContain("FormLabel");
    expect(roleSheetSource).not.toContain('className="textlabel"');
    expect(roleSheetSource).not.toContain(
      'className="text-sm font-medium text-main"'
    );
    expect(roleSheetSource).not.toContain('className="flex flex-col gap-y-5"');
  });
});
