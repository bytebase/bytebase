import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const sectionDir = dirname(fileURLToPath(import.meta.url));

const configurableSections = [
  "AccountSection.tsx",
  "AIAugmentationSection.tsx",
  "AnnouncementSection.tsx",
  "AuditLogSection.tsx",
  "BrandingSection.tsx",
  "GeneralSection.tsx",
  "ProductImprovementSection.tsx",
  "SecuritySection.tsx",
  "SQLEditorSection.tsx",
] as const;

describe("GeneralPage section layout", () => {
  test("uses the shared field group wrapper for configurable section content", () => {
    for (const file of configurableSections) {
      const source = readFileSync(join(sectionDir, file), "utf8");

      expect(source, file).toContain("FormFieldGroup");
      expect(source, file).toContain('className="flex-1 mt-4 lg:px-4 lg:mt-0"');
    }
  });

  test("uses the shared RadioGroup for AccountSection radio controls", () => {
    const source = readFileSync(join(sectionDir, "AccountSection.tsx"), "utf8");

    expect(source).toContain("RadioGroup");
    expect(source).toContain("RadioGroupItem");
    expect(source).not.toContain(`type=${JSON.stringify("radio")}`);
  });
});
