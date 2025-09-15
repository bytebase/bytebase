import { DatabaseChangeMode } from "./proto-es/v1/setting_service_pb";

export type AppFeatures = {
  // Use simple and accurate phrases. Namespace if needed
  "bb.feature.database-change-mode":
    | DatabaseChangeMode.PIPELINE
    | DatabaseChangeMode.EDITOR;
  "bb.feature.hide-help": boolean;
  "bb.feature.hide-quick-start": boolean;
  "bb.feature.hide-trial": boolean;
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
    "bb.feature.sql-editor.sql-check-style": "NOTIFICATION",
  },
});
