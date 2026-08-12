import { router } from "@/app/router";
import { useAppStore } from "@/stores/app";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { type BehaviorMetricName, createBehaviorMetric } from "./behavior";
import { behaviorAnalytics } from "./provider";

type FeatureGateMetricName = Extract<
  BehaviorMetricName,
  "locked feature clicked"
>;

export function captureFeatureGateMetric(
  event: FeatureGateMetricName,
  feature: PlanFeature,
  instance?: Instance | InstanceResource
): void {
  const store = useAppStore.getState();
  const instanceMissingLicense = store.instanceMissingLicense(
    feature,
    instance
  );
  const requiredPlan = store.getMinimumRequiredPlan(feature);

  behaviorAnalytics.captureMetric(
    createBehaviorMetric(event, {
      routeId: router.currentRoute.value.name?.toString(),
      properties: {
        feature: PlanFeature[feature],
        lock_reason: instanceMissingLicense
          ? "instance_license"
          : "subscription_plan",
        required_plan: PlanType[requiredPlan],
      },
    })
  );
}
