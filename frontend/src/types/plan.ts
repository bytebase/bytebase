import { useI18n } from "vue-i18n";
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
  | "bb.feature.rbac"
  | "bb.feature.disallow-signup"
  | "bb.feature.watermark"
  | "bb.feature.audit-log"
  | "bb.feature.issue-advanced-search"
  // Branding
  | "bb.feature.branding"
  // Change Workflow
  | "bb.feature.dba-workflow"
  | "bb.feature.lgtm"
  | "bb.feature.im.approval"
  | "bb.feature.multi-tenancy"
  | "bb.feature.online-migration"
  | "bb.feature.schema-drift"
  | "bb.feature.sql-review"
  | "bb.feature.mybatis-sql-review"
  | "bb.feature.task-schedule-time"
  | "bb.feature.encrypted-secrets"
  | "bb.feature.database-grouping"
  | "bb.feature.schema-template"
  // VCS Integration
  | "bb.feature.vcs-schema-write-back"
  | "bb.feature.vcs-sheet-sync"
  | "bb.feature.vcs-sql-review"
  // Database management
  | "bb.feature.pitr"
  | "bb.feature.read-replica-connection"
  | "bb.feature.instance-ssh-connection"
  | "bb.feature.sync-schema-all-versions"
  | "bb.feature.index-advisor"
  // Policy Control
  | "bb.feature.approval-policy"
  | "bb.feature.backup-policy"
  | "bb.feature.environment-tier-policy"
  | "bb.feature.sensitive-data"
  | "bb.feature.access-control"
  | "bb.feature.custom-approval"
  // Collaboration
  | "bb.feature.shared-sql-script"
  // Plugins
  | "bb.feature.plugin.openai";

export const instanceLimitFeature = new Set<FeatureType>([
  // Change Workflow
  "bb.feature.im.approval",
  "bb.feature.schema-drift",
  "bb.feature.encrypted-secrets",
  "bb.feature.sql-review",
  "bb.feature.task-schedule-time",
  "bb.feature.online-migration",
  // Database Management
  "bb.feature.pitr",
  "bb.feature.read-replica-connection",
  "bb.feature.instance-ssh-connection",
  "bb.feature.sync-schema-all-versions",
  "bb.feature.index-advisor",
  "bb.feature.database-grouping",
  // Policy Control
  "bb.feature.sensitive-data",
  "bb.feature.custom-approval",
  // VCS Integration
  "bb.feature.vcs-sql-review",
  "bb.feature.mybatis-sql-review",
  "bb.feature.vcs-schema-write-back",
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

export type PlanPatch = {
  type: PlanType;
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
  // Plan desc and feature
  title: string;
  featureList: PlanFeature[];
}

export const FEATURE_SECTIONS: { type: string; featureList: string[] }[] =
  planData.categoryList;

export const PLANS: Plan[] = planData.planList.map((raw: Plan) => ({
  ...raw,
  type: planTypeFromJSON(raw.type + 1),
}));

// TODO: it's better to get the count limit from the backend.
export const userCountLimit = new Map<PlanType, number>([
  [PlanType.FREE, 20],
  [PlanType.TEAM, Number.MAX_VALUE],
  [PlanType.ENTERPRISE, Number.MAX_VALUE],
]);

export const instanceCountLimit = new Map<PlanType, number>([
  [PlanType.FREE, 10],
  [PlanType.TEAM, 20],
  [PlanType.ENTERPRISE, Number.MAX_VALUE],
]);

export const getFeatureLocalization = (feature: PlanFeature): PlanFeature => {
  const { t } = useI18n();
  for (const section of FEATURE_SECTIONS) {
    if (new Set(section.featureList).has(feature.type)) {
      const res: PlanFeature = {
        type: t(
          `subscription.feature-sections.${section.type}.features.${feature.type}`
        ),
      };
      if (feature.content) {
        res.content = t(
          `subscription.feature-sections.${section.type}.features.${feature.content}`
        );
      }
      if (feature.tooltip) {
        res.tooltip = t(
          `subscription.feature-sections.${section.type}.features.${feature.tooltip}`
        );
      }
      return res;
    }
  }

  return feature;
};
