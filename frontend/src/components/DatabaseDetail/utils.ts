import { DatabaseLabel, EnvironmentId, InstanceId, ProjectId } from "@/types";

// MySQL only by now
export type CreatePITRDatabaseContext = {
  projectId: ProjectId;
  environmentId: EnvironmentId;
  instanceId: InstanceId;
  databaseName: string;
  characterSet: string;
  collation: string;
  labelList: DatabaseLabel[];
};
