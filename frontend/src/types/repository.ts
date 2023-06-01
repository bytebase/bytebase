export type RepositoryConfig = {
  baseDirectory: string;
  branchFilter: string;
  filePathTemplate: string;
  schemaPathTemplate: string;
  sheetPathTemplate: string;
  enableSQLReviewCI: boolean;
};

export type ExternalRepositoryInfo = {
  // e.g. In GitLab, this is the corresponding project id. e.g. 123
  externalId: string;
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webUrl: string;
};
