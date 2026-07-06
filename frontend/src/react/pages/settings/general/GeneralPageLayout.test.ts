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
  test("uses merged form section layout for configurable sections", () => {
    for (const file of configurableSections) {
      const source = readFileSync(join(sectionDir, file), "utf8");

      expect(source, file).toContain("@/react/components/ui/form");
      expect(source, file).not.toContain("@/react/components/ui/form.v2");
      expect(source, file).toContain("FormSection");
      expect(source, file).toContain("FormFieldGroup");
      expect(source, file).not.toContain("FormSectionHeader");
      expect(source, file).not.toContain("FormSectionTitle");
      expect(source, file).not.toContain("FormSectionContent");
      expect(source, file).not.toContain(
        'className="flex-1 mt-4 lg:px-4 lg:mt-0"'
      );
      expect(source, file).not.toContain('className="py-6 lg:flex"');
    }
  });

  test("uses the shared RadioGroup for AccountSection radio controls", () => {
    const source = readFileSync(join(sectionDir, "AccountSection.tsx"), "utf8");

    expect(source).toContain("RadioGroup");
    expect(source).toContain("RadioGroupItem");
    expect(source).not.toContain(`type=${JSON.stringify("radio")}`);
  });

  test("uses merged form field props for migrated field headings", () => {
    for (const file of ["AnnouncementSection.tsx", "GeneralSection.tsx"]) {
      const source = readFileSync(join(sectionDir, file), "utf8");

      expect(source, file).toContain("<FormField");
      expect(source, file).not.toContain("FormFieldHeader");
      expect(source, file).not.toContain("FormFieldTitle");
      expect(source, file).not.toContain("FormFieldSubtitle");
      expect(source, file).not.toContain(
        '<FormLabel className="text-base font-semibold">'
      );
    }
  });
});
