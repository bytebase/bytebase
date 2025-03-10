import { DatabaseChangeMode } from "./proto/v1/setting_service";

export type AppFeatures = {
  // Use simple and accurate phrases. Namespace if needed
  "bb.feature.default-workspace-view": "CONSOLE" | "EDITOR";
  "bb.feature.database-change-mode":
    | DatabaseChangeMode.PIPELINE
    | DatabaseChangeMode.EDITOR;
  "bb.feature.disallow-navigate-to-console": boolean;
  "bb.feature.hide-banner": boolean;
  "bb.feature.hide-help": boolean;
  "bb.feature.hide-quick-start": boolean;
  "bb.feature.hide-trial": boolean;
  "bb.feature.hide-release-remind": boolean;
  "bb.feature.members.hide-groups": boolean;
  "bb.feature.members.hide-project-roles": boolean;
  "bb.feature.members.hide-service-account": boolean;
  "bb.feature.console.hide-sidebar": boolean;
  "bb.feature.console.hide-header": boolean;
  "bb.feature.console.hide-quick-action": boolean;
  "bb.feature.project.hide-default": boolean;
  "bb.feature.issue.hide-review-actions": boolean;
  "bb.feature.issue.disable-schema-editor": boolean;
  "bb.feature.issue.hide-subscribers": boolean;
  "bb.feature.sql-check.hide-doc-link": boolean;
  "bb.feature.databases.operations": Set<
    | "EDIT-SCHEMA"
    | "CHANGE-DATA"
    | "EXPORT-DATA"
    | "SYNC-SCHEMA"
    | "EDIT-LABELS"
    | "EDIT-ENVIRONMENT"
    | "TRANSFER-OUT"
    | "TRANSFER-IN"
  >;
  "bb.feature.databases.hide-unassigned": boolean;
  "bb.feature.databases.hide-inalterable": boolean;
  "bb.feature.sql-editor.disable-setting": boolean;
  "bb.feature.sql-editor.disallow-share-worksheet": boolean;
  "bb.feature.sql-editor.disallow-batch-query": boolean;
  "bb.feature.sql-editor.disallow-export-query-data": boolean;
  "bb.feature.sql-editor.disallow-request-query": boolean;
  "bb.feature.sql-editor.disallow-sync-schema": boolean;
  "bb.feature.sql-editor.disallow-edit-schema": boolean;
  "bb.feature.sql-editor.hide-bytebase-logo": boolean;
  "bb.feature.sql-editor.hide-profile": boolean;
  "bb.feature.sql-editor.hide-readonly-datasource-hint": boolean;
  "bb.feature.sql-editor.hide-projects": boolean;
  "bb.feature.sql-editor.hide-environments": boolean;
  "bb.feature.sql-editor.hide-advance-instance-features": boolean;
  "bb.feature.sql-editor.sql-check-style": "PREFLIGHT" | "NOTIFICATION";
};

export type AppProfile = {
  embedded: boolean; // Whether the web app is embedded within iframe or not
  features: AppFeatures;
};

export const defaultAppProfile = (): AppProfile => ({
  embedded: false,
  features: {
    "bb.feature.default-workspace-view": "CONSOLE",
    "bb.feature.database-change-mode": DatabaseChangeMode.PIPELINE,
    "bb.feature.disallow-navigate-to-console": false,
    "bb.feature.hide-banner": false,
    "bb.feature.hide-help": false,
    "bb.feature.hide-quick-start": false,
    "bb.feature.hide-release-remind": false,
    "bb.feature.hide-trial": false,
    "bb.feature.members.hide-groups": false,
    "bb.feature.members.hide-project-roles": false,
    "bb.feature.members.hide-service-account": false,
    "bb.feature.console.hide-sidebar": false,
    "bb.feature.console.hide-header": false,
    "bb.feature.console.hide-quick-action": false,
    "bb.feature.project.hide-default": false,
    "bb.feature.issue.hide-review-actions": false,
    "bb.feature.issue.disable-schema-editor": false,
    "bb.feature.issue.hide-subscribers": false,
    "bb.feature.sql-check.hide-doc-link": false,
    "bb.feature.databases.operations": new Set([
      "EDIT-SCHEMA",
      "CHANGE-DATA",
      "EXPORT-DATA",
      "SYNC-SCHEMA",
      "EDIT-LABELS",
      "EDIT-ENVIRONMENT",
      "TRANSFER-OUT",
      "TRANSFER-IN",
    ]),
    "bb.feature.databases.hide-unassigned": false,
    "bb.feature.databases.hide-inalterable": false,
    "bb.feature.sql-editor.disable-setting": false,
    "bb.feature.sql-editor.disallow-share-worksheet": false,
    "bb.feature.sql-editor.disallow-batch-query": false,
    "bb.feature.sql-editor.disallow-export-query-data": false,
    "bb.feature.sql-editor.disallow-request-query": false,
    "bb.feature.sql-editor.disallow-sync-schema": false,
    "bb.feature.sql-editor.disallow-edit-schema": false,
    "bb.feature.sql-editor.hide-bytebase-logo": false,
    "bb.feature.sql-editor.hide-profile": false,
    "bb.feature.sql-editor.hide-readonly-datasource-hint": false,
    "bb.feature.sql-editor.hide-projects": false,
    "bb.feature.sql-editor.hide-environments": false,
    "bb.feature.sql-editor.hide-advance-instance-features": false,
    "bb.feature.sql-editor.sql-check-style": "NOTIFICATION",
  },
});
