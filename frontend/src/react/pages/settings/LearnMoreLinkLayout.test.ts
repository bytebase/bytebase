import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const settingsDir = dirname(fileURLToPath(import.meta.url));

function readSettingsPage(filename: string) {
  return readFileSync(join(settingsDir, filename), "utf8");
}

describe("settings learn more links", () => {
  test("uses LearnMoreLink instead of hand-written common.learn-more anchors", () => {
    const migratedPages = [
      "EnvironmentsPage.tsx",
      "IMPage.tsx",
      "GlobalMaskingPage.tsx",
    ];

    for (const filename of migratedPages) {
      const source = readSettingsPage(filename);

      expect(source).toContain("LearnMoreLink");
      expect(source).not.toContain('{t("common.learn-more")}');
      expect(source).not.toContain('{t("common.learn-more")} &gt;');
    }
  });

  test("adds environment policy learn more links to both page and drawer", () => {
    const source = readSettingsPage("EnvironmentsPage.tsx");

    expect(source.match(/<LearnMoreLink/g)?.length).toBe(4);
    expect(
      source.match(
        /https:\/\/docs\.bytebase\.com\/change-database\/environment-policy\/overview\/\?source=console#environment-tier/g
      )?.length
    ).toBe(2);
    expect(
      source.match(
        /https:\/\/docs\.bytebase\.com\/change-database\/environment-policy\/rollout-policy\/\?source=console/g
      )?.length
    ).toBe(2);
  });
});
