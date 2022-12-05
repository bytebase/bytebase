import { useI18n } from "vue-i18n";
import planData from "./plan.yaml";

// Check api/plan.go to understand what each feature means.
export type FeatureType =
  // Admin & Security
  | "bb.feature.3rd-party-auth"
  | "bb.feature.rbac"
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
  | "bb.feature.vcs-sql-review"
  // Database management
  | "bb.feature.pitr"
  | "bb.feature.read-replica-connection"
  | "bb.feature.sync-schema-all-versions"
  // Policy Control
  | "bb.feature.approval-policy"
  | "bb.feature.backup-policy"
  | "bb.feature.environment-tier-policy"
  | "bb.feature.sensitive-data"
  | "bb.feature.access-control";

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

// A map from a particular feature to the respective enablement of a particular plan.
// The key is the feature type and the value is the [FREE, TEAM, ENTERPRISE] triplet.
// Make sure this is consistent with the matrix in plan.go
//
// TODO: fetch the matrix from the backend instead of duplicating it here or use a JSON/YAML file
// so that it can be shared between frontend/backend.
export const FEATURE_MATRIX: Map<FeatureType, boolean[]> = new Map([
  // Admin & Security
  ["bb.feature.3rd-party-auth", [false, true, true]],
  ["bb.feature.rbac", [false, true, true]],
  // Branding
  ["bb.feature.branding", [false, false, true]],
  // Change Workflow
  ["bb.feature.data-source", [false, false, false]],
  ["bb.feature.dba-workflow", [false, false, true]],
  ["bb.feature.lgtm", [false, false, true]],
  ["bb.feature.im.approval", [false, false, true]],
  ["bb.feature.multi-tenancy", [false, false, true]],
  ["bb.feature.online-migration", [false, true, true]],
  ["bb.feature.schema-drift", [false, true, true]],
  ["bb.feature.sql-review", [false, true, true]],
  ["bb.feature.task-schedule-time", [false, true, true]],
  ["bb.feature.vcs-sql-review", [false, false, true]],
  // Database management
  ["bb.feature.pitr", [false, true, true]],
  ["bb.feature.read-replica-connection", [false, false, true]],
  // This feature type is specifically means that all schema versions can be selected.
  // Sync schema is free to all plans. But in non-enterprise plan, we only show the
  // latest schema version and it's not selectable.
  ["bb.feature.sync-schema-all-versions", [false, false, true]],
  // Policy Control
  ["bb.feature.approval-policy", [false, true, true]],
  ["bb.feature.backup-policy", [false, true, true]],
  ["bb.feature.environment-tier-policy", [false, false, true]],
  ["bb.feature.sensitive-data", [false, false, true]],
  ["bb.feature.access-control", [false, false, true]],
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
