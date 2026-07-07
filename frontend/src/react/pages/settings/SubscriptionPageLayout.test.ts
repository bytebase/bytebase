import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const sectionDir = dirname(fileURLToPath(import.meta.url));

describe("SubscriptionPage layout", () => {
  test("uses shared form primitives for self-hosted subscription fields", () => {
    const source = readFileSync(
      join(sectionDir, "SubscriptionPage.tsx"),
      "utf8"
    );

    expect(source).toContain("FormFieldGroup");
    expect(source).toContain("<FormField");
    expect(source).toContain("LearnMoreLink");
    expect(source).toContain(
      'href="https://www.bytebase.com/pricing?source=console"'
    );
    expect(source).toContain('htmlFor="subscription-workspace-id"');
    expect(source).toContain('id="subscription-workspace-id"');
    expect(source).toContain('htmlFor="subscription-license"');
    expect(source).toContain('id="subscription-license"');
    expect(source).not.toContain(
      '<label className="flex items-center gap-x-2">'
    );
    expect(source).not.toContain(
      'className="mb-3 text-sm text-control-placeholder"'
    );
    expect(source).not.toContain('rel="noopener noreferrer"');
    expect(source).not.toContain('{t("common.learn-more")} &gt;');
  });
});
