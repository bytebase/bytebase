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
