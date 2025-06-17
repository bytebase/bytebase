import { DatabaseChangeMode } from "./proto/v1/setting_service";

export type AppFeatures = {
  // Use simple and accurate phrases. Namespace if needed
  "bb.feature.database-change-mode":
    | DatabaseChangeMode.PIPELINE
    | DatabaseChangeMode.EDITOR;
  "bb.feature.hide-help": boolean;
  "bb.feature.hide-quick-start": boolean;
  "bb.feature.hide-trial": boolean;
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
  "bb.feature.sql-editor.disallow-request-query": boolean;
  "bb.feature.sql-editor.disallow-edit-schema": boolean;
  "bb.feature.sql-editor.enable-setting": boolean;
  "bb.feature.sql-editor.sql-check-style": "PREFLIGHT" | "NOTIFICATION";
};

export type AppProfile = {
  features: AppFeatures;
};

export const defaultAppProfile = (): AppProfile => ({
  features: {
    "bb.feature.database-change-mode": DatabaseChangeMode.PIPELINE,
    "bb.feature.hide-help": false,
    "bb.feature.hide-quick-start": false,
    "bb.feature.hide-trial": false,
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
    "bb.feature.sql-editor.disallow-request-query": false,
    "bb.feature.sql-editor.disallow-edit-schema": false,
    "bb.feature.sql-editor.enable-setting": false,
    "bb.feature.sql-editor.sql-check-style": "NOTIFICATION",
  },
});
