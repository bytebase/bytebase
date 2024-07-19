export type CustomFeatureMatrix = {
  "bb.custom-feature.embedded-in-iframe": boolean;
  "bb.custom-feature.custom-color-scheme": Record<string, string> | undefined;
  "bb.custom-feature.custom-query-datasource": boolean;
  "bb.custom-feature.disallow-navigate-to-console": boolean;
  "bb.custom-feature.disallow-navigate-away-sql-editor": boolean;
  "bb.custom-feature.disallow-share-worksheet": boolean;
  "bb.custom-feature.disallow-export-query-data": boolean;
  "bb.custom-feature.hide-help": boolean;
  "bb.custom-feature.hide-quick-start": boolean;
  "bb.custom-feature.hide-release-remind": boolean;
  "bb.custom-feature.hide-issue-review-actions": boolean;
};

export type CustomFeature = keyof CustomFeatureMatrix;

export const defaultCustomFeatureMatrix = (): CustomFeatureMatrix => ({
  "bb.custom-feature.embedded-in-iframe": false,
  "bb.custom-feature.custom-color-scheme": undefined,
  "bb.custom-feature.custom-query-datasource": false,
  "bb.custom-feature.disallow-navigate-to-console": false,
  "bb.custom-feature.disallow-navigate-away-sql-editor": false,
  "bb.custom-feature.disallow-share-worksheet": false,
  "bb.custom-feature.disallow-export-query-data": false,
  "bb.custom-feature.hide-help": false,
  "bb.custom-feature.hide-quick-start": false,
  "bb.custom-feature.hide-release-remind": false,
  "bb.custom-feature.hide-issue-review-actions": false,
});
