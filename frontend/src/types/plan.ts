import { useI18n } from "vue-i18n";
import planData from "./plan.yaml";

// Check api/plan.go to understand what each feature means.
export type FeatureType =
  // Database management
  | "bb.feature.disaster-recovery-pitr"
  // Change Workflow
  | "bb.feature.sql-review"
  | "bb.feature.schema-drift"
  | "bb.feature.task-schedule-time"
  | "bb.feature.multi-tenancy"
  | "bb.feature.dba-workflow"
  | "bb.feature.data-source"
  | "bb.feature.online-migration"
  // Policy Control
  | "bb.feature.approval-policy"
  | "bb.feature.backup-policy"
  | "bb.feature.environment-tier-policy"
  // Admin & Security
  | "bb.feature.rbac"
  | "bb.feature.3rd-party-auth"
  // Branding
  | "bb.feature.branding";

export enum PlanType {
  FREE = 0,
  TEAM = 1,
  ENTERPRISE = 2,
}

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
  // Change Workflow
  ["bb.feature.sql-review", [false, true, true]],
  ["bb.feature.schema-drift", [false, true, true]],
  ["bb.feature.task-schedule-time", [false, true, true]],
  ["bb.feature.multi-tenancy", [false, true, true]],
  ["bb.feature.dba-workflow", [false, false, true]],
  ["bb.feature.data-source", [false, false, false]],
  ["bb.feature.online-migration", [false, true, true]],
  // Policy Control
  ["bb.feature.approval-policy", [false, true, true]],
  ["bb.feature.backup-policy", [false, true, true]],
  ["bb.feature.environment-tier-policy", [false, false, true]],
  // Admin & Security
  ["bb.feature.rbac", [false, true, true]],
  ["bb.feature.3rd-party-auth", [false, true, true]],
  // Branding
  ["bb.feature.branding", [false, true, true]],
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
