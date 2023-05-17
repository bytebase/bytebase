import { DatabaseLabel, InstanceId } from "@/types";

// MySQL only by now
export type CreatePITRDatabaseContext = {
  projectId: string;
  environmentId: string;
  instanceId: InstanceId;
  databaseName: string;
  characterSet: string;
  collation: string;
  labelList: DatabaseLabel[];
};
