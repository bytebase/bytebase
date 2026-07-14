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

  test("does not add top padding to the first general settings section", () => {
    const source = readFileSync(join(sectionDir, "GeneralSection.tsx"), "utf8");

    expect(source).toContain("style={{ paddingTop: 8 }}");
  });

  test("uses merged form field props for migrated field headings", () => {
    for (const file of configurableSections) {
      const source = readFileSync(join(sectionDir, file), "utf8");

      expect(source, file).toContain("<FormField");
      expect(source, file).not.toContain("FormFieldHeader");
      expect(source, file).not.toContain("FormFieldTitle");
      expect(source, file).not.toContain("FormFieldSubtitle");
      expect(source, file).not.toContain(
        '<FormLabel className="text-base font-semibold">'
      );
      expect(source, file).not.toContain("text-base font-semibold");
      expect(source, file).not.toContain("text-sm font-medium text-control");
    }
  });

  test("keeps domain restriction checkbox text wired to the checkbox", () => {
    const source = readFileSync(
      join(sectionDir, "SecuritySection.tsx"),
      "utf8"
    );
    const membersRestrictionIndex = source.indexOf(
      "settings.general.workspace.domain-restriction.members-restriction.self"
    );
    const membersRestrictionBlock = source.slice(
      Math.max(0, membersRestrictionIndex - 1400),
      membersRestrictionIndex + 800
    );

    expect(membersRestrictionIndex).toBeGreaterThan(0);
    expect(membersRestrictionBlock).toContain(
      '<div className="flex items-start gap-x-2">'
    );
    expect(membersRestrictionBlock).toContain("<Checkbox");
    expect(membersRestrictionBlock).toContain("aria-labelledby");
    expect(membersRestrictionBlock).toContain("aria-describedby");
    expect(membersRestrictionBlock).toContain("toggleDomainRestriction");
    expect(membersRestrictionBlock).toContain(
      "settings.general.workspace.domain-restriction.members-restriction.description"
    );
    expect(membersRestrictionBlock).not.toContain("<label");
    expect(membersRestrictionBlock).not.toContain(
      '<span className="flex items-start gap-x-2">'
    );
  });

  test("wires product intro targets for general settings deep links", () => {
    const aiSource = readFileSync(
      join(sectionDir, "AIAugmentationSection.tsx"),
      "utf8"
    );
    const securitySource = readFileSync(
      join(sectionDir, "SecuritySection.tsx"),
      "utf8"
    );

    expect(aiSource).toContain("AI_ASSISTANT_PRODUCT_INTRO");
    expect(aiSource).toContain("data-product-intro-target");
    expect(aiSource).toContain("useProductIntro");
    expect(securitySource).toContain("DOMAIN_RESTRICTION_PRODUCT_INTRO");
    expect(securitySource).toContain("data-product-intro-target");
    expect(securitySource).toContain("useProductIntro");
  });
});
