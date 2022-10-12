import { useI18n } from "vue-i18n";
import planData from "./plan.yaml";

// Check api/plan.go to understand what each feature means.
export type FeatureType =
  // Database management
  | "bb.feature.disaster-recovery-pitr"
  | "bb.feature.sync-schema"
  // Change Workflow
  | "bb.feature.sql-review"
  | "bb.feature.schema-drift"
  | "bb.feature.task-schedule-time"
  | "bb.feature.multi-tenancy"
  | "bb.feature.dba-workflow"
  | "bb.feature.data-source"
  | "bb.feature.online-migration"
  | "bb.feature.vcs-sql-review"
  // Policy Control
  | "bb.feature.approval-policy"
  | "bb.feature.backup-policy"
  | "bb.feature.environment-tier-policy"
  | "bb.feature.lgtm"
  // Admin & Security
  | "bb.feature.rbac"
  | "bb.feature.3rd-party-auth"
  | "bb.feature.read-replica-connection"
  // Branding
  | "bb.feature.branding";

export enum PlanType {
  FREE = 0,
  TEAM = 1,
  ENTERPRISE = 2,
}

export const planTypeToString = (planType: PlanType): string => {
  switch (planType) {
    case PlanType.FREE:
      return "free";
    case PlanType.TEAM:
      return "team";
    case PlanType.ENTERPRISE:
      return "enterprise";
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
  unitPrice: number;
  trialPrice: number;
  freeInstanceCount: number;
  pricePerInstancePerMonth: number;
  // Plan desc and feature
  title: string;
  featureList: PlanFeature[];
}

// A map from the a particular feature to the respective enablement of a particular plan
// Make sure this is consistent with the matrix in plan.go
//
// TODO: fetch the matrix from the backend instead of duplicating it here or use a JSON/YAML file
// so that it can be shared between frontend/backend.
export const FEATURE_MATRIX: Map<FeatureType, boolean[]> = new Map([
  // Database management
  ["bb.feature.disaster-recovery-pitr", [false, true, true]],
  ["bb.feature.sync-schema", [false, false, true]],
  // Change Workflow
  ["bb.feature.sql-review", [false, true, true]],
  ["bb.feature.schema-drift", [false, true, true]],
  ["bb.feature.task-schedule-time", [false, true, true]],
  ["bb.feature.multi-tenancy", [false, false, true]],
  ["bb.feature.dba-workflow", [false, false, true]],
  ["bb.feature.data-source", [false, false, false]],
  ["bb.feature.online-migration", [false, true, true]],
  ["bb.feature.vcs-sql-review", [false, false, true]],
  // Policy Control
  ["bb.feature.approval-policy", [false, true, true]],
  ["bb.feature.backup-policy", [false, true, true]],
  ["bb.feature.environment-tier-policy", [false, false, true]],
  ["bb.feature.lgtm", [false, false, true]],
  // Admin & Security
  ["bb.feature.rbac", [false, true, true]],
  ["bb.feature.3rd-party-auth", [false, true, true]],
  ["bb.feature.read-replica-connection", [false, false, true]],
  // Branding
  ["bb.feature.branding", [false, false, true]],
]);

export const FEATURE_SECTIONS: { type: string; featureList: string[] }[] =
  planData.categoryList;

export const PLANS: Plan[] = planData.planList;

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
