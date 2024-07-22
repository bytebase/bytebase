export type AppMode = "CONSOLE" | "SQL-EDITOR";

export type AppFeatures = {
  // Use simple and accurate phrases. Namespace if needed
  "bb.feature.embedded-in-iframe": boolean;
  "bb.feature.custom-color-scheme": Record<string, string> | undefined;
  "bb.feature.custom-query-datasource": boolean;
  "bb.feature.disable-kbar": boolean;
  "bb.feature.disable-schema-editor": boolean;
  "bb.feature.database-operations": Set<
    | "EDIT-SCHEMA"
    | "CHANGE-DATA"
    | "EXPORT-DATA"
    | "SYNC-SCHEMA"
    | "EDIT-LABELS"
    | "TRANSFER"
  >;
  "bb.feature.disallow-navigate-to-console": boolean;
  "bb.feature.disallow-share-worksheet": boolean;
  "bb.feature.disallow-export-query-data": boolean;
  "bb.feature.hide-banner": boolean;
  "bb.feature.hide-help": boolean;
  "bb.feature.hide-quick-start": boolean;
  "bb.feature.hide-release-remind": boolean;
  "bb.feature.hide-issue-review-actions": boolean;
  "bb.feature.console.hide-sidebar": boolean;
  "bb.feature.console.hide-header": boolean;
  "bb.feature.console.hide-quick-action": boolean;
  "bb.feature.databases.hide-unassigned": boolean;
  "bb.feature.databases.hide-inalterable": boolean;
};

export type AppProfile = {
  mode: AppMode;
  embedded: boolean; // Whether the web app is embedded within iframe or not
  features: AppFeatures;
};

export const defaultAppProfile = (): AppProfile => ({
  mode: "CONSOLE",
  embedded: false,
  features: {
    "bb.feature.embedded-in-iframe": false,
    "bb.feature.custom-color-scheme": undefined,
    "bb.feature.custom-query-datasource": false,
    "bb.feature.disable-kbar": false,
    "bb.feature.disable-schema-editor": false,
    "bb.feature.database-operations": new Set([
      "EDIT-SCHEMA",
      "CHANGE-DATA",
      "EXPORT-DATA",
      "SYNC-SCHEMA",
      "EDIT-LABELS",
      "TRANSFER",
    ]),
    "bb.feature.disallow-navigate-to-console": false,
    "bb.feature.disallow-share-worksheet": false,
    "bb.feature.disallow-export-query-data": false,
    "bb.feature.hide-banner": false,
    "bb.feature.hide-help": false,
    "bb.feature.hide-quick-start": false,
    "bb.feature.hide-release-remind": false,
    "bb.feature.hide-issue-review-actions": false,
    "bb.feature.console.hide-sidebar": false,
    "bb.feature.console.hide-header": false,
    "bb.feature.console.hide-quick-action": false,
    "bb.feature.databases.hide-unassigned": false,
    "bb.feature.databases.hide-inalterable": false,
  },
});
