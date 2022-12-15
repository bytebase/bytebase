export type ServerInfo = {
  version: string;
  gitCommit: string;
  readonly: boolean;
  demo: boolean;
  demoName: string;
  externalUrl: string;
  needAdminSetup: boolean;
  startedTs: number;
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
  lastest?: Release;
  ignoreRemindModalTillNextRelease: boolean;
  // The next check timestamp in milliseconds.
  nextCheckTs: number;
};
