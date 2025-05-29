import {
  PlanType,
  planTypeFromJSON,
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
  | "bb.feature.multi-tenancy"
  | "bb.feature.online-migration"
  | "bb.feature.schema-drift"
  | "bb.feature.sql-review"
  | "bb.feature.task-schedule-time"
  | "bb.feature.encrypted-secrets"
  | "bb.feature.database-grouping"
  | "bb.feature.schema-template"
  | "bb.feature.issue-project-setting"
  // Database management
  | "bb.feature.read-replica-connection"
  | "bb.feature.custom-instance-synchronization"
  | "bb.feature.instance-ssh-connection"
  | "bb.feature.sync-schema-all-versions"
  | "bb.feature.index-advisor"
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
  // Change Workflow
  "bb.feature.schema-drift",
  "bb.feature.encrypted-secrets",
  "bb.feature.task-schedule-time",
  "bb.feature.online-migration",
  // Database Management
  "bb.feature.read-replica-connection",
  "bb.feature.instance-ssh-connection",
  "bb.feature.custom-instance-synchronization",
  "bb.feature.sync-schema-all-versions",
  "bb.feature.index-advisor",
  "bb.feature.database-grouping",
  // Policy Control
  "bb.feature.sensitive-data",
  "bb.feature.rollout-policy",
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

interface PlanFeature {
  type: string;
  content?: string;
  tooltip?: string;
}

export interface Plan {
  // Plan meta data
  type: PlanType;
  trialDays: number;
  trialPrice: number;
  unitPrice: number;
  pricePerSeatPerMonth: number;
  pricePerInstancePerMonth: number;
  maximumSeatCount: number;
  maximumInstanceCount: number;
  // Plan desc and feature
  title: string;
  featureList: PlanFeature[];
}

export const PLANS: Plan[] = planData.planList.map((raw: Plan) => ({
  ...raw,
  type: planTypeFromJSON(raw.type + 1),
}));
