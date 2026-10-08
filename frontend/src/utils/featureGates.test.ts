// @vitest-environment node
import { expect, test, vi } from "vitest";

test("the gate follows the dev build flag", async () => {
  vi.resetModules();
  vi.doMock("./util", () => ({ isDev: () => false }));
  const disabled = await import("./featureGates");
  expect(disabled.sqlReviewV2Enabled()).toBe(false);

  vi.resetModules();
  vi.doMock("./util", () => ({ isDev: () => true }));
  const enabled = await import("./featureGates");
  expect(enabled.sqlReviewV2Enabled()).toBe(true);

  vi.doUnmock("./util");
  vi.resetModules();
});

// Vitest reports DEV, so every other suite exercises the feature as an engineer
// on the dev server sees it. A release bundle is the opposite case, and only the
// suites that stub the gate cover it.
test("the suite runs with the gate on, matching the dev server", async () => {
  const { sqlReviewV2Enabled } = await import("./featureGates");
  expect(import.meta.env.DEV).toBe(true);
  expect(sqlReviewV2Enabled()).toBe(true);
});
