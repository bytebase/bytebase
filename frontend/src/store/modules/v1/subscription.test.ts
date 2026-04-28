import { create } from "@bufbuild/protobuf";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
  SubscriptionSchema,
} from "@/types/proto-es/v1/subscription_service_pb";

vi.mock("@/connect", () => ({
  subscriptionServiceClientConnect: {},
}));

vi.mock("@/types", () => ({
  hasFeature: vi.fn(() => true),
  hasInstanceFeature: vi.fn(
    (_plan: PlanType, _feature: PlanFeature, activated: boolean) => activated
  ),
  getDateForPbTimestampProtoEs: vi.fn(() => new Date()),
  getMinimumRequiredPlan: vi.fn(() => PlanType.TEAM),
  getTimeForPbTimestampProtoEs: vi.fn(() => Date.now()),
  instanceLimitFeature: new Set<PlanFeature>([PlanFeature.FEATURE_DATA_MASKING]),
  PLANS: [
    {
      type: PlanType.ENTERPRISE,
      maximumInstanceCount: -1,
      maximumSeatCount: -1,
    },
  ],
}));

vi.mock("@/utils/datetime", () => ({
  formatAbsoluteDateTime: vi.fn(() => ""),
}));

let useSubscriptionV1Store: typeof import("./subscription").useSubscriptionV1Store;

describe("useSubscriptionV1Store unified instance license", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    setActivePinia(createPinia());
    ({ useSubscriptionV1Store } = await import("./subscription"));
  });

  test("computes unified mode from effective limits", () => {
    const store = useSubscriptionV1Store();
    store.setSubscription(
      create(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        instances: 10,
        activeInstances: 10,
      })
    );

    expect(store.hasUnifiedInstanceLicense).toBe(true);

    store.setSubscription(
      create(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        instances: 50,
        activeInstances: 20,
      })
    );

    expect(store.hasUnifiedInstanceLicense).toBe(false);
  });

  test("does not report missing instance license in unified mode", () => {
    const store = useSubscriptionV1Store();
    store.setSubscription(
      create(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        instances: 10,
        activeInstances: 10,
      })
    );

    const inactiveInstance = create(InstanceSchema, {
      name: "instances/prod",
      title: "prod",
      activation: false,
    });

    expect(
      store.instanceMissingLicense(
        PlanFeature.FEATURE_DATA_MASKING,
        inactiveInstance
      )
    ).toBe(false);
  });
});
