import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { captureFeatureGateMetric } from "./feature-gate";

const mocks = vi.hoisted(() => ({
  captureMetric: vi.fn(),
  instanceMissingLicense: vi.fn(),
  getMinimumRequiredPlan: vi.fn(),
}));

vi.mock("@/app/router", () => ({
  router: { currentRoute: { value: { name: "workspace.profile" } } },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      instanceMissingLicense: mocks.instanceMissingLicense,
      getMinimumRequiredPlan: mocks.getMinimumRequiredPlan,
    }),
  },
}));

vi.mock("./provider", () => ({
  behaviorAnalytics: { captureMetric: mocks.captureMetric },
}));

describe("feature gate analytics", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.instanceMissingLicense.mockReturnValue(false);
    mocks.getMinimumRequiredPlan.mockReturnValue(PlanType.TEAM);
  });

  test("captures safe subscription gate context", () => {
    captureFeatureGateMetric(
      "locked feature clicked",
      PlanFeature.FEATURE_TWO_FA
    );

    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "locked feature clicked",
      properties: {
        route_id: "workspace.profile",
        feature: "FEATURE_TWO_FA",
        lock_reason: "subscription_plan",
        required_plan: "TEAM",
      },
    });
  });

  test("distinguishes an instance license gate", () => {
    mocks.instanceMissingLicense.mockReturnValue(true);

    captureFeatureGateMetric(
      "locked feature clicked",
      PlanFeature.FEATURE_DATA_MASKING,
      { name: "instances/test" } as never
    );

    expect(mocks.captureMetric).toHaveBeenCalledWith(
      expect.objectContaining({
        properties: expect.objectContaining({
          lock_reason: "instance_license",
        }),
      })
    );
  });
});
