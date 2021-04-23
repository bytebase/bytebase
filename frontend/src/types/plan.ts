export type FeatureType =
  // Support Owner and DBA role at the workspace level
  | "bytebase.admin"
  // Support DBA workflow, including
  // 1. Developers can't create database directly, they need to do this via a request db issue.
  // 2. Allow developers to submit troubleshooting ticket.
  | "bytebase.dba-workflow"
  // Support defining extra data source for a database and exposing the related data source UI.
  | "bytebase.data-source";

export enum PlanType {
  FREE = 0,
  TEAM = 1,
  ENTERPRISE = 2,
}
