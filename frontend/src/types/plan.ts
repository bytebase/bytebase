import {
  PlanFeature,
  PlanType,
  type PlanLimitConfig,
} from "@/types/proto-es/v1/subscription_service_pb";
import planData from "./plan.yaml";

// Convert YAML data to proper types
export const PLANS: PlanLimitConfig[] = planData.plans.map((plan: any) => ({
  ...plan,
  type: PlanType[plan.type as keyof typeof PlanType],
  features: plan.features.map(
    (f: string) => PlanFeature[f as keyof typeof PlanFeature]
  ),
}));

// Create a plan feature matrix from the YAML data
const planFeatureMatrix = new Map<PlanType, Set<PlanFeature>>();
// Instance-limited features that require activated instances
export const instanceLimitFeature = new Set<PlanFeature>();

// Initialize the feature matrix and instance features from plan data
PLANS.forEach((plan) => {
  planFeatureMatrix.set(plan.type, new Set(plan.features));
});
planData.instanceFeatures.forEach((feature: string) => {
  instanceLimitFeature.add(PlanFeature[feature as keyof typeof PlanFeature]);
});

// Helper function to check if a plan has a feature
export const planHasFeature = (
  plan: PlanType,
  feature: PlanFeature
): boolean => {
  const planFeatures = planFeatureMatrix.get(plan);
  return planFeatures?.has(feature) ?? false;
};

// Helper function to get minimum required plan for a feature
export const getMinimumRequiredPlan = (feature: PlanFeature): PlanType => {
  const planOrder = [PlanType.FREE, PlanType.TEAM, PlanType.ENTERPRISE];
  for (const plan of planOrder) {
    if (planHasFeature(plan, feature)) {
      return plan;
    }
  }
  return PlanType.ENTERPRISE;
};

// Helper function to check if a feature is available for a plan
export const hasFeature = (plan: PlanType, feature: PlanFeature): boolean => {
  return planHasFeature(plan, feature);
};

// Helper function to check instance features
export const hasInstanceFeature = (
  plan: PlanType,
  feature: PlanFeature,
  instanceActivated = true
): boolean => {
  if (!hasFeature(plan, feature)) {
    return false;
  }

  // For FREE plan, don't check instance activation
  if (plan === PlanType.FREE) {
    return true;
  }

  // For instance-limited features, check activation
  if (instanceLimitFeature.has(feature)) {
    return instanceActivated;
  }

  return true;
};
