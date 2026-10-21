import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({ isDev: false }));

vi.mock("@/utils/featureGates", () => ({
  isSaaSOrDev: (isSaaSMode: boolean) => isSaaSMode || mocks.isDev,
}));

import { PROJECT_V1_ROUTE_DETAIL, PROJECT_V1_ROUTE_ISSUES } from "./handles";
import { buildProjectEntryRoute } from "./projectEntryRoute";

beforeEach(() => {
  mocks.isDev = false;
});

describe("projectEntryRoute", () => {
  test("builds the SaaS project landing route", () => {
    expect(buildProjectEntryRoute("orders", true)).toEqual({
      name: PROJECT_V1_ROUTE_DETAIL,
      params: { projectId: "orders" },
    });
  });

  test("keeps the self-hosted project entry on Issues", () => {
    expect(buildProjectEntryRoute("orders", false)).toEqual({
      name: PROJECT_V1_ROUTE_ISSUES,
      params: { projectId: "orders" },
    });
  });

  test("enables the project landing in development", () => {
    mocks.isDev = true;

    expect(buildProjectEntryRoute("orders", false)).toEqual({
      name: PROJECT_V1_ROUTE_DETAIL,
      params: { projectId: "orders" },
    });
  });
});
