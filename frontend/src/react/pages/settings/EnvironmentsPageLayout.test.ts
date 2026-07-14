import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const settingsDir = dirname(fileURLToPath(import.meta.url));

const readEnvironmentsPage = () =>
  readFileSync(join(settingsDir, "EnvironmentsPage.tsx"), "utf8");

describe("EnvironmentsPage layout", () => {
  test("keeps the environment tier feature badge spaced from both titles", () => {
    const source = readEnvironmentsPage();

    expect(
      source.match(/<span className="inline-flex items-center gap-x-1">/g)
        ?.length
    ).toBeGreaterThanOrEqual(2);
    expect(source).toContain('{t("policy.environment-tier.name")}');
    expect(
      source.match(
        /<FeatureBadge\s+feature=\{PlanFeature\.FEATURE_ENVIRONMENT_TIERS\}\s*\/>/g
      )?.length
    ).toBeGreaterThanOrEqual(2);
  });
});
