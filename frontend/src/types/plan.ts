export type FeatureType =
  // Support Owner and DBA role at the workspace level
  | "bb.admin"
  // Support DBA workflow, including
  // 1. Developers can't create database directly, they need to do this via a request db issue.
  // 2. Allow developers to submit troubleshooting ticket.
  | "bb.dba-workflow"
  // Support defining extra data source for a database and exposing the related data source UI.
  | "bb.data-source";

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
