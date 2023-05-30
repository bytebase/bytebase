// MySQL only by now
export type CreatePITRDatabaseContext = {
  projectId: string;
  environmentId: string;
  instanceId: string;
  databaseName: string;
  characterSet: string;
  collation: string;
  labels: Record<string, string>;
};
