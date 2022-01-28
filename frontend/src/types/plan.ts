// Check api/plan.go to understand what each feature means.
export type FeatureType =
  // Change Workflow
  | "bb.feature.backward-compatibility"
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
  | "bb.feature.3rd-party-login";

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
  description: string;
  features: { id: string; content?: string }[];
}
