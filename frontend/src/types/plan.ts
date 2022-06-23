// Check api/plan.go to understand what each feature means.
export type FeatureType =
  // Change Workflow
  | "bb.feature.schema-review-policy"
  | "bb.feature.schema-drift"
  | "bb.feature.task-schedule-time"
  | "bb.feature.multi-tenancy"
  | "bb.feature.dba-workflow"
  | "bb.feature.data-source"
  // Policy Control
  | "bb.feature.approval-policy"
  | "bb.feature.backup-policy"
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
  features: { id: string; content?: string; tooltip?: string }[];
}

// A map from the a particular feature to the respective enablement of a particular plan
// Make sure this is consistent with the matrix in plan.go
//
// TODO: fetch the matrix from the backend instead of duplicating it here or use a JSON/YAML file
// so that it can be shared between frontend/backend.
export const FEATURE_MATRIX: Map<FeatureType, boolean[]> = new Map([
  // Change Workflow
  ["bb.feature.schema-review-policy", [false, true, true]],
  ["bb.feature.schema-drift", [false, true, true]],
  ["bb.feature.task-schedule-time", [false, true, true]],
  ["bb.feature.multi-tenancy", [false, true, true]],
  ["bb.feature.dba-workflow", [false, false, true]],
  ["bb.feature.data-source", [false, false, false]],
  // Policy Control
  ["bb.feature.approval-policy", [false, true, true]],
  ["bb.feature.backup-policy", [false, true, true]],
  // Admin & Security
  ["bb.feature.rbac", [false, true, true]],
  ["bb.feature.3rd-party-auth", [false, true, true]],
  // Branding
  ["bb.feature.branding", [false, true, true]],
]);

export const FEATURE_SECTIONS = [
  {
    id: "database-management",
    features: [
      "instance-count",
      "schema-change",
      "migration-history",
      "sql-editor",
      "disaster-recovery",
      "archiving",
      "sql-check",
      "anomaly-detection",
      "schedule-change",
      "review-and-backup-policy",
      "tenancy",
    ],
  },
  {
    id: "collaboration",
    features: [
      "ui-based-sql-review",
      "vsc-workflow",
      "shareable-query-link",
      "im-integration",
      "inbox-notification",
    ],
  },
  {
    id: "admin-and-security",
    features: [
      "activity-log",
      "rbac",
      "3rd-party-auth",
      "sync-members-from-vcs",
    ],
  },
  {
    id: "branding",
    features: ["branding-logo"],
  },
];

export const FREE_PLAN: Plan = {
  // Plan meta data
  type: PlanType.FREE,
  trialDays: 0,
  unitPrice: 0,
  trialPrice: 0,
  freeInstanceCount: 1,
  pricePerInstancePerMonth: 0,
  // Plan desc and feature
  title: "free",
  features: [
    {
      id: "instance-count",
      content:
        "subscription.feature-sections.database-management.features.instance-upto-5",
    },
    { id: "schema-change" },
    { id: "migration-history" },
    { id: "sql-editor" },
    {
      id: "disaster-recovery",
      content:
        "subscription.feature-sections.database-management.features.disaster-recovery-basic",
      tooltip:
        "subscription.feature-sections.database-management.features.disaster-recovery-basic-tooltip",
    },
    { id: "archiving" },
    {
      id: "sql-check",
      content:
        "subscription.feature-sections.database-management.features.sql-check-basic",
      tooltip:
        "subscription.feature-sections.database-management.features.sql-check-basic-tooltip",
    },
    {
      id: "anomaly-detection",
      content:
        "subscription.feature-sections.database-management.features.anomaly-detection-basic",
      tooltip:
        "subscription.feature-sections.database-management.features.anomaly-detection-basic-tooltip",
    },
    { id: "ui-based-sql-review" },
    { id: "vsc-workflow" },
    { id: "shareable-query-link" },
    { id: "im-integration" },
    { id: "inbox-notification" },
    { id: "activity-log" },
  ],
};

export const TEAM_PLAN: Plan = {
  // Plan meta data
  type: PlanType.TEAM,
  trialDays: 14,
  unitPrice: 1740,
  trialPrice: 0,
  freeInstanceCount: 5,
  pricePerInstancePerMonth: 79,
  // Plan desc and feature
  title: "team",
  features: [
    {
      id: "instance-count",
      content:
        "subscription.feature-sections.database-management.features.instance-minimum-5",
    },
    { id: "schema-change" },
    { id: "migration-history" },
    { id: "sql-editor" },
    {
      id: "disaster-recovery",
      content:
        "subscription.feature-sections.database-management.features.disaster-recovery-advanced",
      tooltip:
        "subscription.feature-sections.database-management.features.disaster-recovery-advanced-tooltip",
    },
    { id: "archiving" },
    {
      id: "sql-check",
      content:
        "subscription.feature-sections.database-management.features.sql-check-advanced",
      tooltip:
        "subscription.feature-sections.database-management.features.sql-check-advanced-tooltip",
    },
    {
      id: "anomaly-detection",
      content:
        "subscription.feature-sections.database-management.features.anomaly-detection-advanced",
      tooltip:
        "subscription.feature-sections.database-management.features.anomaly-detection-advanced-tooltip",
    },
    { id: "schedule-change" },
    { id: "review-and-backup-policy" },
    { id: "ui-based-sql-review" },
    { id: "vsc-workflow" },
    { id: "shareable-query-link" },
    { id: "im-integration" },
    { id: "inbox-notification" },
    { id: "activity-log" },
    { id: "rbac" },
    { id: "3rd-party-auth" },
    { id: "sync-members-from-vcs" },
    { id: "branding-logo" },
  ],
};

export const ENTERPRISE_PLAN: Plan = {
  // Plan meta data
  type: PlanType.ENTERPRISE,
  trialDays: 0,
  unitPrice: 0,
  trialPrice: 0,
  freeInstanceCount: 5,
  pricePerInstancePerMonth: 199,
  // Plan desc and feature
  title: "enterprise",
  features: [
    {
      id: "instance-count",
      content:
        "subscription.feature-sections.database-management.features.instance-customized",
    },
    { id: "schema-change" },
    { id: "migration-history" },
    { id: "sql-editor" },
    {
      id: "disaster-recovery",
      content:
        "subscription.feature-sections.database-management.features.disaster-recovery-advanced",
      tooltip:
        "subscription.feature-sections.database-management.features.disaster-recovery-advanced-tooltip",
    },
    { id: "archiving" },
    {
      id: "sql-check",
      content:
        "subscription.feature-sections.database-management.features.sql-check-advanced",
      tooltip:
        "subscription.feature-sections.database-management.features.sql-check-advanced-tooltip",
    },
    {
      id: "anomaly-detection",
      content:
        "subscription.feature-sections.database-management.features.anomaly-detection-advanced",
      tooltip:
        "subscription.feature-sections.database-management.features.anomaly-detection-advanced-tooltip",
    },
    { id: "schedule-change" },
    { id: "review-and-backup-policy" },
    { id: "tenancy" },
    { id: "ui-based-sql-review" },
    { id: "vsc-workflow" },
    { id: "shareable-query-link" },
    { id: "im-integration" },
    { id: "inbox-notification" },
    { id: "activity-log" },
    { id: "rbac" },
    { id: "3rd-party-auth" },
    { id: "sync-members-from-vcs" },
    { id: "branding-logo" },
  ],
};
