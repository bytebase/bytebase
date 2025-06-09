import {
  PlanType,
  type PlanConfig,
  type PlanLimitConfig,
} from "@/types/proto/v1/subscription_service";
import planData from "./plan.yaml";

// Check api/plan.go to understand what each feature means.
export type FeatureType =
  // General
  | "bb.feature.custom-role"
  // Admin & Security
  | "bb.feature.sso"
  | "bb.feature.2fa"
  | "bb.feature.secure-token"
  | "bb.feature.password-restriction"
  | "bb.feature.rbac"
  | "bb.feature.disallow-signup"
  | "bb.feature.disallow-password-signin"
  | "bb.feature.domain-restriction"
  | "bb.feature.watermark"
  | "bb.feature.audit-log"
  | "bb.feature.issue-advanced-search"
  | "bb.feature.announcement"
  | "bb.feature.external-secret-manager"
  | "bb.feature.directory-sync"
  // Branding
  | "bb.feature.branding"
  // Change Workflow
  | "bb.feature.dba-workflow"
  | "bb.feature.schema-drift"
  | "bb.feature.sql-review"
  | "bb.feature.encrypted-secrets"
  | "bb.feature.database-grouping"
  | "bb.feature.schema-template"
  | "bb.feature.issue-project-setting"
  // Database management
  | "bb.feature.read-replica-connection"
  | "bb.feature.custom-instance-synchronization"
  | "bb.feature.sync-schema-all-versions"
  // Policy Control
  | "bb.feature.rollout-policy"
  | "bb.feature.environment-tier-policy"
  | "bb.feature.sensitive-data"
  | "bb.feature.access-control"
  | "bb.feature.custom-approval"
  // Efficiency
  | "bb.feature.batch-query"
  // Collaboration
  | "bb.feature.shared-sql-script"
  // Plugins
  | "bb.feature.ai-assistant"
  // Instance count limit
  | "bb.feature.instance-count"
  // User count limit
  | "bb.feature.user-count";

export const instanceLimitFeature = new Set<FeatureType>([
  "bb.feature.encrypted-secrets",
  "bb.feature.read-replica-connection",
  "bb.feature.sensitive-data",
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
