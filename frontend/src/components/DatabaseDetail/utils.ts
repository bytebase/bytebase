import { DatabaseLabel, InstanceId, ProjectId } from "@/types";

// MySQL only by now
export type CreatePITRDatabaseContext = {
  projectId: ProjectId;
  environmentId: number;
  instanceId: InstanceId;
  databaseName: string;
  characterSet: string;
  collation: string;
  labelList: DatabaseLabel[];
};
