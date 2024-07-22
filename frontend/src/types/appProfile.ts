export type AppMode = "CONSOLE" | "SQL-EDITOR";

export type AppFeatures = {
  "bb.feature.embedded-in-iframe": boolean;
  "bb.feature.custom-color-scheme": Record<string, string> | undefined;
  "bb.feature.custom-query-datasource": boolean;
  "bb.feature.disallow-navigate-to-console": boolean;
  "bb.feature.disallow-share-worksheet": boolean;
  "bb.feature.disallow-export-query-data": boolean;
  "bb.feature.hide-help": boolean;
  "bb.feature.hide-quick-start": boolean;
  "bb.feature.hide-release-remind": boolean;
  "bb.feature.hide-issue-review-actions": boolean;
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
    "bb.feature.disallow-navigate-to-console": false,
    "bb.feature.disallow-share-worksheet": false,
    "bb.feature.disallow-export-query-data": false,
    "bb.feature.hide-help": false,
    "bb.feature.hide-quick-start": false,
    "bb.feature.hide-release-remind": false,
    "bb.feature.hide-issue-review-actions": false,
  },
});
