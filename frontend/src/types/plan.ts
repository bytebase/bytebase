import {
  PlanType,
  type PlanConfig,
  type PlanLimitConfig,
  PlanLimitConfig_Feature,
} from "@/types/proto/v1/subscription_service";
import planData from "./plan.yaml";

// Use proto Feature enum as the primary feature type
export type Feature = PlanLimitConfig_Feature;

// Instance-limited features that require activated instances
export const instanceLimitFeature = new Set<PlanLimitConfig_Feature>([
  PlanLimitConfig_Feature.DATABASE_SECRET_VARIABLES,
  PlanLimitConfig_Feature.INSTANCE_READ_ONLY_CONNECTION,
  PlanLimitConfig_Feature.DATA_MASKING,
]);

export const planTypeToString = (planType: PlanType): string => {
  switch (planType) {
    case PlanType.FREE:
      return "free";
    case PlanType.TEAM:
      return "team";
    case PlanType.ENTERPRISE:
      return "enterprise";
    default:
      return "";
  }
};

// Re-export proto types for convenience
export type { PlanConfig, PlanLimitConfig };

export const PLAN_CONFIG: PlanConfig = planData;
export const PLANS: PlanLimitConfig[] = planData.plans;

// Create a plan feature matrix from the YAML data
export const PLAN_FEATURE_MATRIX = new Map<PlanType, Set<PlanLimitConfig_Feature>>();

// Initialize the feature matrix from plan data
PLANS.forEach((plan) => {
  PLAN_FEATURE_MATRIX.set(plan.type, new Set(plan.features));
});

// Helper function to check if a plan has a feature
export const planHasFeature = (plan: PlanType, feature: PlanLimitConfig_Feature): boolean => {
  const planFeatures = PLAN_FEATURE_MATRIX.get(plan);
  return planFeatures?.has(feature) ?? false;
};

// Helper function to get minimum required plan for a feature
export const getMinimumRequiredPlan = (feature: PlanLimitConfig_Feature): PlanType => {
  const planOrder = [PlanType.FREE, PlanType.TEAM, PlanType.ENTERPRISE];
  for (const plan of planOrder) {
    if (planHasFeature(plan, feature)) {
      return plan;
    }
  }
  return PlanType.ENTERPRISE;
};

// Helper function to check if a feature is available for a plan
export const hasFeature = (plan: PlanType, feature: PlanLimitConfig_Feature): boolean => {
  return planHasFeature(plan, feature);
};

// Helper function to check instance features
export const hasInstanceFeature = (plan: PlanType, feature: PlanLimitConfig_Feature, instanceActivated = true): boolean => {
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
