export type Repository = {
  // e.g. In GitLab, this is the corresponding project id.
  externalId: string;
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webURL: string;
};
