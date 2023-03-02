export type ServerInfo = {
  version: string;
  gitCommit: string;
  readonly: boolean;
  saas: boolean;
  demoName: string;
  externalUrl: string;
  needAdminSetup: boolean;
  startedTs: number;
  disallowSignup: boolean;
};

export type Release = {
  draft: boolean;
  prerelease: boolean;
  name: string;
  tag_name: string;
  html_url: string;
  body: string;
  published_at: string;
};

export type ReleaseInfo = {
  latest?: Release;
  ignoreRemindModalTillNextRelease: boolean;
  // The next check timestamp in milliseconds.
  nextCheckTs: number;
};
