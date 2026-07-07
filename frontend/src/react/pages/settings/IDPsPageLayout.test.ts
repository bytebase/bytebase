import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const sectionDir = dirname(fileURLToPath(import.meta.url));

const getFunctionSource = (source: string, name: string, nextMarker: string) =>
  source.slice(source.indexOf(`function ${name}`), source.indexOf(nextMarker));

describe("IDPsPage create SSO layout", () => {
  test("uses shared form primitives in the create SSO form steps", () => {
    const source = readFileSync(join(sectionDir, "IDPsPage.tsx"), "utf8");
    const providerConfigSource = getFunctionSource(
      source,
      "ProviderConfigForm",
      "// ============================================================\n// FieldMappingForm"
    );
    const fieldMappingSource = getFunctionSource(
      source,
      "FieldMappingForm",
      "// ============================================================\n// CreateWizardDrawer"
    );
    const createWizardSource = getFunctionSource(
      source,
      "CreateWizardDrawer",
      "// ============================================================\n// IDPsPage"
    );

    for (const formSource of [
      providerConfigSource,
      fieldMappingSource,
      createWizardSource,
    ]) {
      expect(formSource).toContain("FormField");
      expect(formSource).not.toContain(
        'className="block text-base font-semibold text-gray-800'
      );
    }

    expect(createWizardSource).toContain("<FormFieldGroup");
    expect(createWizardSource).toContain("FormTitle");
    expect(createWizardSource).not.toContain("FormLabel");
    expect(createWizardSource).toContain('id="sso-create-name"');
    expect(createWizardSource).toContain('id="sso-create-domain"');
    expect(providerConfigSource).toContain("FormFieldGroup");
    expect(providerConfigSource).toContain("FormTitle");
    expect(providerConfigSource).not.toContain("FormLabel");
    expect(fieldMappingSource).toContain("FormFieldGroup");
  });
});
