import { useI18n } from "vue-i18n";
import planData from "./plan.yaml";
import { PlanType, planTypeFromJSON } from "@/types/proto/store/subscription";

// Check api/plan.go to understand what each feature means.
export type FeatureType =
  // General
  | "bb.feature.custom-role"
  // Admin & Security
  | "bb.feature.sso"
  | "bb.feature.2fa"
  | "bb.feature.rbac"
  | "bb.feature.disallow-signup"
  | "bb.feature.watermark"
  | "bb.feature.audit-log"
  // Branding
  | "bb.feature.branding"
  // Change Workflow
  | "bb.feature.data-source"
  | "bb.feature.dba-workflow"
  | "bb.feature.lgtm"
  | "bb.feature.im.approval"
  | "bb.feature.multi-tenancy"
  | "bb.feature.online-migration"
  | "bb.feature.schema-drift"
  | "bb.feature.sql-review"
  | "bb.feature.task-schedule-time"
  | "bb.feature.encrypted-secrets"
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
