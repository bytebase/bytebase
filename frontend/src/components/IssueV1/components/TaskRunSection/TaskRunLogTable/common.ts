import type {
  TaskRunLogEntry_CommandExecute,
  TaskRunLogEntry_SchemaDump,
  TaskRunLogEntry_Type,
} from "@/types/proto/v1/rollout_service";

export type FlattenLogEntry = {
  batch: number;
  serial: number;
  type: TaskRunLogEntry_Type;
  startTime?: Date;
  endTime?: Date;
  schemaDump?: TaskRunLogEntry_SchemaDump;
  commandExecute?: {
    raw: TaskRunLogEntry_CommandExecute;
    commandIndex: number;
    affectedRows?: number;
  };
};
