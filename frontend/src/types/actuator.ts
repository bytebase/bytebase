export type ServerInfo = {
  version: string;
  readonly: boolean;
  demo: boolean;
  host: string;
  port: string;
  needAdminSetup: boolean;
  startedTs: number;
};
