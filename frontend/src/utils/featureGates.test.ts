import { afterEach, expect, test, vi } from "vitest";
import { STORAGE_KEY_SQL_REVIEW_VERSION } from "./storage-keys";

afterEach(() => {
  localStorage.clear();
});

test("a release bundle turns the gate on for Bytebase Cloud only", async () => {
  vi.resetModules();
  vi.doMock("./util", () => ({ isDev: () => false }));
  const { sqlReviewV2Enabled } = await import("./featureGates");
  const { useAppStore } = await import("@/stores/app");

  useAppStore.setState({ serverInfo: undefined });
  expect(sqlReviewV2Enabled()).toBe(false);

  useAppStore.setState({ serverInfo: { saas: false } as never });
  expect(sqlReviewV2Enabled()).toBe(false);

  useAppStore.setState({ serverInfo: { saas: true } as never });
  expect(sqlReviewV2Enabled()).toBe(true);

  // Cloud has no V1/V2 switch to read.
  localStorage.setItem(STORAGE_KEY_SQL_REVIEW_VERSION, "v1");
  expect(sqlReviewV2Enabled()).toBe(true);

  useAppStore.setState({ serverInfo: undefined });
  vi.doUnmock("./util");
  vi.resetModules();
});

test("a dev build turns the gate on whether or not it runs as Cloud", async () => {
  vi.resetModules();
  vi.doMock("./util", () => ({ isDev: () => true }));
  const { sqlReviewV2Enabled } = await import("./featureGates");
  const { useAppStore } = await import("@/stores/app");

  useAppStore.setState({ serverInfo: { saas: false } as never });
  expect(sqlReviewV2Enabled()).toBe(true);

  useAppStore.setState({ serverInfo: undefined });
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

test("a dev browser switched to V1 turns the gate off until it switches back", async () => {
  const { devSqlReviewVersion, setDevSqlReviewVersion, sqlReviewV2Enabled } =
    await import("./featureGates");
  expect(devSqlReviewVersion()).toBe("v2");

  setDevSqlReviewVersion("v1");
  expect(devSqlReviewVersion()).toBe("v1");
  expect(sqlReviewV2Enabled()).toBe(false);

  setDevSqlReviewVersion("v2");
  expect(sqlReviewV2Enabled()).toBe(true);
});
