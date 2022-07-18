export type ServerInfo = {
  version: string;
  gitCommit: string;
  readonly: boolean;
  demo: boolean;
  demoName: string;
  host: string;
  port: string;
  needAdminSetup: boolean;
  startedTs: number;
};
